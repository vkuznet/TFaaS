package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var page []byte
	var err error
	fmt.Println("loading", r.URL.Path)
	if r.URL.Path != "/" {
		path := strings.TrimLeft(r.URL.Path, "/")
		page, err = ioutil.ReadFile(path)
	} else {
		page, err = ioutil.ReadFile("view-browser.html")
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(page)
}

func main() {
	http.HandleFunc("/", RequestHandler)
	http.ListenAndServe(":8888", nil)
}
