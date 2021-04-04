package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"
)

// helper function to return current version
func info() string {
	goVersion := runtime.Version()
	tstamp := time.Now()
	return fmt.Sprintf("Build: git={{VERSION}} go=%s date=%s", goVersion, tstamp)
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
	server(config)

}
