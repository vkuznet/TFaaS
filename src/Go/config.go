package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

// TFaaS configuration
var _config Configuration

// Configuration stores dbs configuration parameters
type Configuration struct {
	Port             int    `json:"port"`        // dbs port number
	ModelDir         string `json:"modelDir"`    // location of model directory
	StaticDir        string `json:"staticDir"`   // speficy static dir location
	ConfigProto      string `json:"configProto"` // TF config proto file to use
	Base             string `json:"base"`        // dbs base path
	LogFile          string `json:"logFile"`     // log file
	Verbose          int    `json:"verbose"`     // verbosity level
	ServerKey        string `json:"serverKey"`   // server key for https
	ServerCrt        string `json:"serverCrt"`   // server certificate for https
	CacheLimit       int    `json:"cacheLimit"`  // number of TFModels to keep in cache
	LimiterPeriod    string `json:"rate"`        // github.com/ulule/limiter rate value
	PrintMonitRecord bool   `json:"monitRecord"` // print monit record on stdout
}

// String returns string representation of server configuration
func (c *Configuration) String() string {
	return fmt.Sprintf("config port=%d modelDir=%s staticDir=%s base=%s proto=%s verbose=%d log=%s crt=%s key=%s rate=%s", c.Port, c.ModelDir, c.StaticDir, c.Base, c.ConfigProto, c.Verbose, c.LogFile, c.ServerCrt, c.ServerKey, c.LimiterPeriod)
}

// helper function to parse configuration file
func parseConfig(configFile string) error {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Println("configFile", configFile, err)
		return err
	}
	err = json.Unmarshal(data, &_config)
	if err != nil {
		log.Println("configFile", configFile, err)
		return err
	}
	if _config.LimiterPeriod == "" {
		_config.LimiterPeriod = "100-S"
	}
	log.Println(_config.String())
	return nil
}
