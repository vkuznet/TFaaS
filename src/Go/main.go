package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"

	logs "github.com/sirupsen/logrus"
)

// VERBOSE controls verbosity of the server
var VERBOSE int

// Auth represents flag to use authentication or not
var Auth string

// Configuration stores dbs configuration parameters
type Configuration struct {
	Port         int    `json:"port"`         // dbs port number
	Auth         string `json:"auth"`         // use authentication or not
	ModelDir     string `json:"modelDir"`     // location of model directory
	ModelName    string `json:"model"`        // name of the model to use
	ModelLabels  string `json:"labels"`       // name of labels file to use
	InputNode    string `json:"inputNode"`    // TF input node name to use
	OutputNode   string `json:"outputNode"`   // TF output node name to use
	ConfigProto  string `json:"configProto"`  // TF config proto file to use
	Base         string `json:"base"`         // dbs base path
	LogFormatter string `json:"logFormatter"` // log formatter
	Verbose      int    `json:"verbose"`      // verbosity level
	ServerKey    string `json:"serverKey"`    // server key for https
	ServerCrt    string `json:"serverCrt"`    // server certificate for https
}

// String returns string representation of server configuration
func (c *Configuration) String() string {
	return fmt.Sprintf("<Config port=%d dir=%s base=%s auth=%s model=%s labels=%s inputNode=%s outptuNode=%s configProt=%s verbose=%d log=%s crt=%s key=%s>", c.Port, c.ModelDir, c.Base, c.Auth, c.ModelName, c.ModelLabels, c.InputNode, c.OutputNode, c.ConfigProto, c.Verbose, c.LogFormatter, c.ServerCrt, c.ServerKey)
}

// Params returns string representation of server parameters
func (c *Configuration) Params() string {
	return fmt.Sprintf("<Params model=%s labels=%s inputNode=%s outptuNode=%s configProt=%s verbose=%d log=%s>", c.ModelName, c.ModelLabels, c.InputNode, c.OutputNode, c.ConfigProto, c.Verbose, c.LogFormatter)
}

func main() {
	var config string
	flag.StringVar(&config, "config", "config.json", "configuration file for our server")
	flag.Parse()

	var err error
	_client = httpClient()
	err = parseConfig(config)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Fatal("Unable to parse config")
	}

	// create session options from given config TF proto file
	_sessionOptions = readConfigProto(_config.ConfigProto)
	_inputNode = _config.InputNode
	_outputNode = _config.OutputNode
	_configProto = _config.ConfigProto
	_modelDir = _config.ModelDir
	_modelName = _config.ModelName
	_modelLabels = _config.ModelLabels
	Auth = _config.Auth

	if _modelName != "" {
		err = loadModel(_modelName, _modelLabels)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Error":  err,
				"Model":  _modelName,
				"Labels": _modelLabels,
			}).Error("unable to open TF model")
		}
		logs.WithFields(logs.Fields{
			"Auth":        Auth,
			"Model":       _modelName,
			"Labels":      _modelLabels,
			"InputNode":   _inputNode,
			"OutputNode":  _outputNode,
			"ConfigProto": _configProto,
		}).Info("serving TF model")
	} else {
		logs.Warn("No model file is supplied, will unable to run inference")
	}

	http.Handle("/models/", http.StripPrefix("/models/", http.FileServer(http.Dir(_modelDir))))
	http.HandleFunc("/", AuthHandler)
	addr := fmt.Sprintf(":%d", _config.Port)
	if _config.ServerKey != "" && _config.ServerCrt != "" {
		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				ClientAuth: tls.RequestClientCert,
			},
		}
		if _, err := os.Open(_config.ServerKey); err != nil {
			logs.WithFields(logs.Fields{
				"Error": err,
				"File":  _config.ServerKey,
			}).Error("unable to open server key file")
		}
		if _, err := os.Open(_config.ServerCrt); err != nil {
			logs.WithFields(logs.Fields{
				"Error": err,
				"File":  _config.ServerCrt,
			}).Error("unable to open server cert file")
		}
		logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTPs server")
		err = server.ListenAndServeTLS(_config.ServerCrt, _config.ServerKey)
	} else {
		logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTP server")
		err = http.ListenAndServe(addr, nil)
	}
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Fatal("ListenAndServe: ")
	}
}
