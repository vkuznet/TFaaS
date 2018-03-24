package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"time"

	logs "github.com/sirupsen/logrus"
	"github.com/vkuznet/x509proxy"
)

// UserDNs structure holds information about user DNs
type UserDNs struct {
	DNs  []string
	Time time.Time
}

// global variable which we initialize once
var _userDNs UserDNs

// global HTTP client
var _client *http.Client

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
		if Auth == "true" {
			logs.Fatal("Neither proxy or user certs are found, use X509_USER_PROXY/X509_USER_KEY/X509_USER_CERT to set them up or run with -auth false")
		}
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

// httpClient provides HTTP client
func httpClient() *http.Client {
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

// helper function to parse configuration file
func parseConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		logs.WithFields(logs.Fields{"configFile": configFile}).Fatal("Unable to read", err)
		return err
	}
	err = json.Unmarshal(data, &_config)
	if err != nil {
		logs.WithFields(logs.Fields{"configFile": configFile}).Fatal("Unable to parse", err)
		return err
	}
	logs.Info(_config.String())
	return nil
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
	userDN := UserDN(r)
	match := InList(userDN, _userDNs.DNs)
	if !match {
		logs.WithFields(logs.Fields{
			"User DN": userDN,
		}).Error("Auth userDN not found in SiteDB")
	}
	return match
}
