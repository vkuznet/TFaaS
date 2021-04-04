package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// VERBOSE controls verbosity of the server
var VERBOSE int

// Time0 represents initial time when we start the server
var Time0 time.Time

// global variables
var (
	_header, _footer, _tmplDir string
)

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

func basePath(s string) string {
	if _config.Base != "" {
		if strings.HasPrefix(s, "/") {
			s = strings.Replace(s, "/", "", 1)
		}
		if strings.HasPrefix(_config.Base, "/") {
			return fmt.Sprintf("%s/%s", _config.Base, s)
		}
		return fmt.Sprintf("/%s/%s", _config.Base, s)
	}
	return s
}

func handlers() *mux.Router {
	router := mux.NewRouter()

	// visible routes
	router.HandleFunc(basePath("/upload"), UploadHandler).Methods("POST")
	router.HandleFunc(basePath("/delete"), DeleteHandler).Methods("DELETE")
	router.HandleFunc(basePath("/data"), DataHandler).Methods("GET")
	router.HandleFunc(basePath("/json"), PredictHandler).Methods("POST")
	router.HandleFunc(basePath("/proto"), PredictProtobufHandler).Methods("POST")
	router.HandleFunc(basePath("/image"), ImageHandler).Methods("POST")
	router.HandleFunc(basePath("/params"), ParamsHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/models"), ModelsHandler).Methods("GET")
	router.HandleFunc(basePath("/status"), StatusHandler).Methods("GET")
	router.HandleFunc(basePath("/netron/"), NetronHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/netron/{.*}"), NetronHandler).Methods("GET", "POST")
	router.HandleFunc(basePath("/"), DefaultHandler).Methods("GET")

	/* for future use
	// for all requests perform first auth/authz action
	router.Use(authMiddleware)
	// validate all input parameters
	router.Use(validateMiddleware)

	// use limiter middleware to slow down clients
	router.Use(limitMiddleware)

	*/

	return router
}

// server represents main web server
func server(config string) {
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
	VERBOSE = _config.Verbose

	// define our handlers
	sdir := _config.StaticDir
	if sdir == "" {
		path, _ := os.Getwd()
		sdir = fmt.Sprintf("%s/static", path)
	}
	_tmplDir = fmt.Sprintf("%s/templates", sdir)

	// static handlers
	base := _config.Base
	for _, name := range []string{"js", "css", "images", "download", "tempaltes"} {
		m := fmt.Sprintf("%s/%s/", base, name)
		if base == "" || base == "/" {
			m = fmt.Sprintf("/%s/", name)
		}
		d := fmt.Sprintf("%s/%s", sdir, name)
		if name == "download" {
			d = _config.ModelDir
		}
		log.Printf("static '%s' => '%s'\n", m, http.Dir(d))
		http.Handle(m, http.StripPrefix(m, http.FileServer(http.Dir(d))))
	}
	http.Handle(basePath("/"), handlers())

	// setup templates
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Base"] = _config.Base
	tmplData["Content"] = fmt.Sprintf("Hello from TFaaS")
	tmplData["Version"] = info()
	tmplData["Models"], _ = TFModels()
	_header = templates.Header(_tmplDir, tmplData)
	_footer = templates.Footer(_tmplDir, tmplData)

	// start web server
	addr := fmt.Sprintf(":%d", _config.Port)
	_, e1 := os.Stat(_config.ServerCrt)
	_, e2 := os.Stat(_config.ServerKey)
	if e1 == nil && e2 == nil {
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
