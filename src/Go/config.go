package main

import "fmt"

// Configuration stores dbs configuration parameters
type Configuration struct {
	Port        int    `json:"port"`        // dbs port number
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
	return fmt.Sprintf("<Config port=%d modelDir=%s staticDir=%s base=%s configProt=%s verbose=%d log=%s crt=%s key=%s>", c.Port, c.ModelDir, c.StaticDir, c.Base, c.ConfigProto, c.Verbose, c.LogFile, c.ServerCrt, c.ServerKey)
}
