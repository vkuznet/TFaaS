package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	logs "github.com/sirupsen/logrus"
	"github.com/vkuznet/x509proxy"
)

// global variable which we initialize once
var _userDNs []string

// global HTTP client
var _client = HttpClient()

// global client's x509 certificates
var _certs []tls.Certificate

// client X509 certificates
func tlsCerts() ([]tls.Certificate, error) {
	if len(_certs) != 0 {
		return _certs, nil // use cached certs
	}
	uproxy := os.Getenv("X509_USER_PROXY")
	uckey := os.Getenv("X509_USER_KEY")
	ucert := os.Getenv("X509_USER_CERT")

	// check if /tmp/x509up_u$UID exists, if so setup X509_USER_PROXY env
	u, err := user.Current()
	if err == nil {
		fname := fmt.Sprintf("/tmp/x509up_u%s", u.Uid)
		if _, err := os.Stat(fname); err == nil {
			uproxy = fname
		}
	}
	if uproxy == "" && uckey == "" { // user doesn't have neither proxy or user certs
		logs.Warn("Neither proxy or user certs are found, to proceed use auth=false option otherwise setup X509 environment")
		return nil, nil
	}
	if uproxy != "" {
		// use local implementation of LoadX409KeyPair instead of tls one
		x509cert, err := x509proxy.LoadX509Proxy(uproxy)
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy X509 proxy set by X509_USER_PROXY: %v", err)
		}
		_certs = []tls.Certificate{x509cert}
		return _certs, nil
	}
	x509cert, err := tls.LoadX509KeyPair(ucert, uckey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user X509 certificate: %v", err)
	}
	_certs = []tls.Certificate{x509cert}
	return _certs, nil
}

// HttpClient provides HTTP client
func HttpClient() *http.Client {
	// get X509 certs
	certs, err := tlsCerts()
	if err != nil {
		panic(err.Error())
	}
	if len(certs) == 0 {
		return &http.Client{}
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{Certificates: certs,
			InsecureSkipVerify: true},
	}
	return &http.Client{Transport: tr}
}

func userDNs() []string {
	var out []string
	rurl := "https://cmsweb.cern.ch/sitedb/data/prod/people"
	req, err1 := http.NewRequest("GET", rurl, nil)
	if err1 != nil {
		logs.WithFields(logs.Fields{
			"Error": err1,
		}).Error("Unable to make GET request")
		return out
	}
	req.Header.Add("Accept", "*/*")
	resp, err2 := _client.Do(req)
	if err2 != nil {
		logs.WithFields(logs.Fields{
			"Error": err2,
		}).Error("Unable to place HTTP request")
		return out
	}
	defer resp.Body.Close()
	data, err3 := ioutil.ReadAll(resp.Body)
	if err3 != nil {
		logs.WithFields(logs.Fields{
			"Error": err3,
		}).Error("Unable to make GET request")
		return out
	}
	var rec map[string]interface{}
	err4 := json.Unmarshal(data, &rec)
	if err4 != nil {
		logs.WithFields(logs.Fields{
			"Error": err4,
		}).Error("Unable to unmarshal response")
		return out
	}
	desc := rec["desc"].(map[string]interface{})
	headers := desc["columns"].([]interface{})
	var idx int
	for i, h := range headers {
		if h.(string) == "dn" {
			idx = i
			break
		}
	}
	values := rec["result"].([]interface{})
	for _, item := range values {
		val := item.([]interface{})
		v := val[idx]
		if v != nil {
			out = append(out, v.(string))
		}
	}
	return out
}

// InList helper function to check item in a list
func InList(a string, list []string) bool {
	check := 0
	for _, b := range list {
		if b == a {
			check += 1
		}
	}
	if check != 0 {
		return true
	}
	return false
}

// UserDN function parses user Distinguished Name (DN) from client's HTTP request
func UserDN(r *http.Request) string {
	var names []interface{}
	for _, cert := range r.TLS.PeerCertificates {
		for _, name := range cert.Subject.Names {
			switch v := name.Value.(type) {
			case string:
				names = append(names, v)
			}
		}
	}
	parts := names[:7]
	return fmt.Sprintf("/DC=%s/DC=%s/OU=%s/OU=%s/CN=%s/CN=%s/CN=%s", parts...)
}

// custom logic for CMS authentication, users may implement their own logic here
func auth(r *http.Request) bool {

	if len(_userDNs) == 0 {
		_userDNs = userDNs()
	}
	userDN := UserDN(r)
	match := InList(userDN, _userDNs)
	if !match {
		logs.WithFields(logs.Fields{
			"User DN": userDN,
		}).Error("Auth userDN not found in SiteDB")
	}
	return match
}

// DataHandler authenticate incoming requests and route them to appropriate handler
func DataHandler(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	if files, ok := args["model"]; ok {
		fname := files[0]
		if _, err := os.Stat(fname); !os.IsNotExist(err) {
			var fin *os.File
			fin, err := os.Open(fname)
			if err != nil {
				logs.WithFields(logs.Fields{
					"Error": err,
					"File":  fname,
				}).Error("unable to open model file")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// we don't need to WriteHeader here since it is handled by http.ServeContent
			http.ServeContent(w, r, fname, time.Now(), fin)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.WriteHeader(http.StatusBadRequest)
}

// DefaultHandler authenticate incoming requests and route them to appropriate handler
func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	msg := fmt.Sprintf("Hello %s", UserDN(r))
	w.Write([]byte(msg))
}

// AuthHandler authenticate incoming requests and route them to appropriate handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	// check if server started with hkey file (auth is required)
	status := auth(r)
	if !status {
		msg := "You are not allowed to access this resource"
		http.Error(w, msg, http.StatusForbidden)
		return
	}
	arr := strings.Split(r.URL.Path, "/")
	path := arr[len(arr)-1]
	switch path {
	case "data":
		DataHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
}

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "models", "local directory to serve by this server")
	flag.Parse()

	http.Handle("/models/", http.StripPrefix("/models/", http.FileServer(http.Dir(dir))))
	http.HandleFunc("/", AuthHandler)
	server := &http.Server{
		Addr: ":8083",
		TLSConfig: &tls.Config{
			ClientAuth: tls.RequestClientCert,
		},
	}
	server.ListenAndServeTLS("server.crt", "server.key")
}
