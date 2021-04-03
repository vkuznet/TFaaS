package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// VERBOSE controls verbosity of the server
var VERBOSE int

// Auth represents flag to use authentication or not
var Auth string

// Time0 represents initial time when we start the server
var Time0 time.Time

// global variables
var (
	_header, _footer, _tmplDir string
)

// helper function to produce UTC time prefixed output
func utcMsg(data []byte) string {
	//     return fmt.Sprintf("[" + time.Now().String() + "] " + string(data))
	s := string(data)
	v, e := url.QueryUnescape(s)
	if e == nil {
		return v
	}
	return s
}

// custom rotate logger
type rotateLogWriter struct {
	RotateLogs *rotatelogs.RotateLogs
}

func (w rotateLogWriter) Write(data []byte) (int, error) {
	return w.RotateLogs.Write([]byte(utcMsg(data)))
}

// Configuration stores dbs configuration parameters
type Configuration struct {
	Port        int    `json:"port"`        // dbs port number
	Auth        string `json:"auth"`        // use authentication or not
	ModelDir    string `json:"modelDir"`    // location of model directory
	StaticDir   string `json:"staticDir"`   // speficy static dir location
	ConfigProto string `json:"configProto"` // TF config proto file to use
	Base        string `json:"base"`        // dbs base path
	LogFile     string `json:"logFile"`     // log file
	Verbose     int    `json:"verbose"`     // verbosity level
	ServerKey   string `json:"serverKey"`   // server key for https
	ServerCrt   string `json:"serverCrt"`   // server certificate for https
	UpdateDNs   int    `json:"updateDNs"`   // interval in minutes to update user DNs
	CacheLimit  int    `json:"cacheLimit"`  // number of TFModels to keep in cache
}

// String returns string representation of server configuration
func (c *Configuration) String() string {
	return fmt.Sprintf("<Config port=%d modelDir=%s staticDir=%s base=%s auth=%s configProt=%s verbose=%d log=%s crt=%s key=%s>", c.Port, c.ModelDir, c.StaticDir, c.Base, c.Auth, c.ConfigProto, c.Verbose, c.LogFile, c.ServerCrt, c.ServerKey)
}

// helper function to return current version
func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("Build: git={{VERSION}} go=%s date=%s", goVersion, tstamp)
}

// Memory contains details about memory information
type Memory struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"usedPercent"`
}

// Mem keeps memory information
type Mem struct {
	Virtual Memory
	Swap    Memory
}

func main() {
	var config string
	flag.StringVar(&config, "config", "config.json", "configuration file for our server")
	var version bool
	flag.BoolVar(&version, "version", false, "Show version")
	flag.Parse()

	if version {
		fmt.Println(info())
		os.Exit(0)
	}

	Time0 = time.Now()

	var err error
	_client = httpClient()
	err = parseConfig(config)
	if err != nil {
		log.Println("unable to parse config", err)
	}

	// setup config
	if _config.LogFile != "" {
		logName := _config.LogFile + "-%Y%m%d"
		hostname, err := os.Hostname()
		if err == nil {
			logName = _config.LogFile + "-" + hostname + "-%Y%m%d"
		}
		rl, err := rotatelogs.New(logName)
		if err == nil {
			rotlogs := rotateLogWriter{RotateLogs: rl}
			log.SetOutput(rotlogs)
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		} else {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
		}
	} else {
		// log time, filename, and line number
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// create session options from given config TF proto file
	_sessionOptions = readConfigProto(_config.ConfigProto) // default session options
	cacheLimit := _config.CacheLimit
	if cacheLimit == 0 {
		cacheLimit = 10 // default number of models to keep in cache
	}
	_cache = TFCache{Models: make(map[string]TFCacheEntry), Limit: cacheLimit}
	Auth = _config.Auth // set if we gonna use auth or not
	VERBOSE = _config.Verbose

	// define our handlers
	base := _config.Base
	sdir := _config.StaticDir
	if sdir == "" {
		path, _ := os.Getwd()
		sdir = fmt.Sprintf("%s/static", path)
	}
	tmplDir := fmt.Sprintf("%s/templates", sdir)
	cssDir := fmt.Sprintf("%s/css", sdir)
	jsDir := fmt.Sprintf("%s/js", sdir)
	imgDir := fmt.Sprintf("%s/images", sdir)
	_tmplDir = tmplDir
	http.Handle(base+"/css/", http.StripPrefix(base+"/css/", http.FileServer(http.Dir(cssDir))))
	http.Handle(base+"/js/", http.StripPrefix(base+"/js/", http.FileServer(http.Dir(jsDir))))
	http.Handle(base+"/images/", http.StripPrefix(base+"/images/", http.FileServer(http.Dir(imgDir))))
	http.Handle(base+"/download/", http.StripPrefix(base+"/download/", http.FileServer(http.Dir(_config.ModelDir))))
	http.HandleFunc(base+"/", AuthHandler)

	// setup templates
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Base"] = _config.Base
	tmplData["Content"] = fmt.Sprintf("Hello from TFaaS")
	tmplData["Version"] = info()
	tmplData["Models"], _ = TFModels()
	_header = templates.Header(tmplDir, tmplData)
	_footer = templates.Footer(tmplDir, tmplData)

	// start web server
	addr := fmt.Sprintf(":%d", _config.Port)
	_, e1 := os.Stat(_config.ServerCrt)
	_, e2 := os.Stat(_config.ServerKey)
	if e1 == nil && e2 == nil {

		if Auth == "true" {
			// init userDNs and update it periodically
			_userDNs = UserDNs{DNs: userDNs(), Time: time.Now()}
			go func() {
				for {
					interval := _config.UpdateDNs
					if interval == 0 {
						interval = 60
					}
					d := time.Duration(interval) * time.Minute
					log.Println("userDNs are updated", time.Now())
					time.Sleep(d) // sleep for next iteration
					_userDNs = UserDNs{DNs: userDNs(), Time: time.Now()}
				}
			}()
		}

		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				ClientAuth: tls.RequestClientCert,
			},
		}
		if _, err := os.Open(_config.ServerKey); err != nil {
			log.Println("unable to open server key file", _config.ServerKey, err)
		}
		if _, err := os.Open(_config.ServerCrt); err != nil {
			log.Println("unable to open server cert file", _config.ServerCrt, err)
		}
		log.Println("starting HTTPs server", addr)
		err = server.ListenAndServeTLS(_config.ServerCrt, _config.ServerKey)
	} else {
		log.Println("starting HTTP server", addr)
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}
