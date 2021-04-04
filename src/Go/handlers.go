package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	tf "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/golang/protobuf/proto"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	tfaaspb "github.com/vkuznet/TFaaS/tfaaspb"
)

// TotalGetRequests counts total number of GET requests received by the server
var TotalGetRequests uint64

// TotalPostRequests counts total number of POST requests received by the server
var TotalPostRequests uint64

// helper function to provide response
func responseError(w http.ResponseWriter, msg string, err error, code int) {
	log.Println("ERROR", msg, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// helper function to provide response in JSON data format
func responseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

//
// HTTP handlers, GET methods
//

// DataHandler authenticate incoming requests and route them to appropriate handler
func DataHandler(w http.ResponseWriter, r *http.Request) {
	args := r.URL.Query()
	if files, ok := args["model"]; ok {
		fname := files[0]
		if _, err := os.Stat(fname); !os.IsNotExist(err) {
			var fin *os.File
			fin, err := os.Open(fname)
			if err != nil {
				responseError(w, fmt.Sprintf("unable to open file: %s", fname), err, http.StatusInternalServerError)
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

// ImageHandler send prediction from TF ML model
func ImageHandler(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "POST") {
		responseError(w, fmt.Sprintf("wrong HTTP method: %v", r.Method), nil, http.StatusMethodNotAllowed)
		return
	}
	model := r.FormValue("model")
	if model == "" {
		msg := fmt.Sprintf("unable to read %s model", model)
		responseError(w, msg, nil, http.StatusInternalServerError)
		return
	}
	// read image model
	tfm, err := _cache.get(model)
	if err != nil {
		responseError(w, "unable to get image model from the cache", err, http.StatusInternalServerError)
		return
	}

	// Read image
	imageFile, header, err := r.FormFile("image")
	fileName := header.Filename
	imageName := strings.Split(fileName, ".")
	if err != nil {
		responseError(w, "unable to read image", err, http.StatusInternalServerError)
		return
	}
	defer imageFile.Close()
	var imageBuffer bytes.Buffer
	// Copy image data to a buffer
	io.Copy(&imageBuffer, imageFile)

	// Make tensor
	tensor, err := makeTensorFromImage(&imageBuffer, imageName[:1][0])
	if err != nil {
		responseError(w, "Invalid image", err, http.StatusBadRequest)
		return
	}

	// Run inference
	session, err := tf.NewSession(tfm.Graph, _sessionOptions)
	if err != nil {
		responseError(w, "Unable to create new session", err, http.StatusInternalServerError)
		return
	}
	defer session.Close()
	if VERBOSE > 0 {
		devices, err := session.ListDevices()
		if err == nil {
			log.Println("devices", devices)
		} else {
			log.Println("node availability", err)
		}
	}
	output, err := session.Run(
		map[tf.Output]*tf.Tensor{
			tfm.Graph.Operation(tfm.Params.InputNode).Output(0): tensor,
		},
		[]tf.Output{
			tfm.Graph.Operation(tfm.Params.OutputNode).Output(0),
		},
		nil)
	if err != nil {
		responseError(w, "Could not run inference", err, http.StatusInternalServerError)
		return
	}
	// our model probabilities
	probs := output[0].Value().([][]float32)[0]

	// make prediction response
	topN := 5
	if len(tfm.Labels) < topN {
		topN = len(tfm.Labels)
	}
	responseJSON(w, ClassifyResult{
		Filename: fileName,
		Labels:   findBestLabels(tfm.Labels, probs, topN),
	})
}

// PredictProtobufHandler send prediction from TF ML model
func PredictProtobufHandler(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "POST") {
		responseError(w, fmt.Sprintf("wrong HTTP method: %v", r.Method), nil, http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		responseError(w, "unable to read incoming data", err, http.StatusInternalServerError)
		return
	}
	// example how to unmarshal Row message
	recs := &tfaaspb.Row{}
	if err := proto.Unmarshal(body, recs); err != nil {
		responseError(w, "unable to unmarshal Row", err, http.StatusInternalServerError)
		return
	}
	if VERBOSE > 0 {
		log.Println("received", recs)
	}

	// convert tfaaspb.Row into Row
	var keys []string
	var values []float32
	for _, k := range recs.Key {
		keys = append(keys, k)
	}
	for _, v := range recs.Value {
		values = append(values, v)
	}
	records := &Row{Keys: keys, Values: values, Model: recs.Model}

	// generate predictions
	probs, err := makePredictions(records)
	if err != nil {
		responseError(w, "unable to make predictions", err, http.StatusInternalServerError)
		return
	}

	if VERBOSE > 0 {
		log.Println("response inputs", records, "probs", probs)
	}

	// wrap our probabilities into Predictions class
	var objects []*tfaaspb.Class
	for _, p := range probs {
		objects = append(objects, &tfaaspb.Class{Probability: float32(p)})
	}
	pobj := &tfaaspb.Predictions{Prediction: objects}
	out, err := proto.Marshal(pobj)
	if err != nil {
		responseError(w, "unable to marshal data", err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

// PredictHandler send prediction from TF ML model
func PredictHandler(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "POST") {
		responseError(w, fmt.Sprintf("wrong HTTP method: %v", r.Method), nil, http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		responseError(w, "unable to read incoming data", err, http.StatusInternalServerError)
		return
	}
	// unmarshal incoming JSON message into Row data structure
	recs := &Row{}
	if err := json.Unmarshal(body, recs); err != nil {
		responseError(w, "unable to unmarshal Row", err, http.StatusInternalServerError)
		return
	}
	if VERBOSE > 0 {
		log.Println("received", recs)
	}

	// generate predictions
	probs, err := makePredictions(recs)
	if err != nil {
		responseError(w, "PredictHandler: unable to make predictions", err, http.StatusInternalServerError)
		return
	}
	responseJSON(w, probs)
}

// POST methods

// GzipReader struct to handle GZip'ed content of HTTP requests
type GzipReader struct {
	*gzip.Reader
	io.Closer
}

// helper function to close gzip reader
func (gz GzipReader) Close() error {
	return gz.Closer.Close()
}

// UploadHandler uploads TF models into the server
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.FormValue("model") == "model" {
		// we received request for upload via form values
		UploadFormHandler(w, r)
		return
	}
	UploadBundleHandler(w, r)
}

// UploadBundleHandler uploads TF models into the server
func UploadBundleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		msg := "wrong HTTP method"
		responseError(w, msg, nil, http.StatusMethodNotAllowed)
		return
	}
	var err error
	var bundle []byte
	defer r.Body.Close()
	if r.Header.Get("Content-Encoding") == "gzip" {
		r.Header.Del("Content-Length")
		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			msg := "unable to get gzip reader"
			responseError(w, msg, err, http.StatusInternalServerError)
			return
		}
		body := GzipReader{reader, r.Body}
		bundle, err = ioutil.ReadAll(body)
	} else {
		bundle, err = ioutil.ReadAll(r.Body)
	}
	if err != nil {
		msg := "unable to read body"
		responseError(w, msg, err, http.StatusInternalServerError)
		return
	}
	//     fname := fmt.Sprintf("/tmp/bundle.tar")
	fname := fmt.Sprintf("%s/bundle.tar", os.TempDir())
	defer os.Remove(fname)
	err = ioutil.WriteFile(fname, bundle, 0600)
	if err != nil {
		msg := fmt.Sprintf("unable to write %s", fname)
		responseError(w, msg, err, http.StatusInternalServerError)
		return
	}
	err = Untar(fname, _config.ModelDir)
	if err != nil {
		msg := fmt.Sprintf("unable to untar %s", fname)
		responseError(w, msg, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// UploadFormHandler uploads TF models into the server via form key-value pairs
func UploadFormHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		msg := "wrong HTTP method"
		responseError(w, msg, nil, http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if VERBOSE > 0 {
		log.Println("UploadHandler", r.Header)
	}
	ctype := r.Header.Get("Content-Encoding")
	var mkey, path string
	var params TFParams
	for _, name := range []string{"name", "params", "model", "labels"} {
		emsg := fmt.Sprintf("request does not provide %s", name)
		if name == "name" {
			mkey = r.FormValue(name)
			if mkey == "" {
				responseError(w, emsg, nil, http.StatusInternalServerError)
				return
			}
			path = fmt.Sprintf("%s/%s", _config.ModelDir, mkey)
			// create requested area for TF model
			err := os.MkdirAll(path, 0744)
			if err != nil {
				msg := fmt.Sprintf("unable to create %s", path)
				responseError(w, msg, err, http.StatusInternalServerError)
				return
			}
			continue
		}
		// read other parameters which represent files
		modelFile, header, err := r.FormFile(name)
		if err != nil {
			responseError(w, emsg, err, http.StatusInternalServerError)
			return
		}
		defer modelFile.Close()

		// prepare file name to write to
		arr := strings.Split(header.Filename, "/")
		fname := arr[len(arr)-1]
		if name == "params" && fname != "params.json" {
			fname = "params.json"
			msg := fmt.Sprintf("store as %s", fname)
			log.Println("file", header.Filename, msg)
		}
		fileName := fmt.Sprintf("%s/%s", path, fname)

		// read data from request and write it out to our local file
		data, err := ioutil.ReadAll(modelFile)
		if err != nil {
			responseError(w, "unable to read model file", err, http.StatusInternalServerError)
			return
		}

		// read TF parameters
		if name == "params" {
			err = json.Unmarshal(data, &params)
			if err != nil {
				responseError(w, "unable to unmarshal TF parameters", err, http.StatusInternalServerError)
				return
			}
			if params.TimeStamp == "" {
				params.TimeStamp = time.Now().String()
			}
			if mkey != params.Name {
				msg := fmt.Sprintf("mismatch of mkey=%s and TFParam.Name=%s", mkey, params.Name)
				responseError(w, msg, err, http.StatusInternalServerError)
				return
			}
			log.Println("TF model parameters", params.String())
		}

		if ctype == "base64" && name == "model" {
			var newData []byte
			newData, err = base64.StdEncoding.DecodeString(string(data))
			if err != nil {
				responseError(w, "unable to decode input data", err, http.StatusInternalServerError)
				return
			}
			err = ioutil.WriteFile(fileName, newData, 0644)
			if err != nil {
				responseError(w, "unable to write file", err, http.StatusInternalServerError)
				return
			}

		} else {
			// write out content to our store
			err = ioutil.WriteFile(fileName, data, 0644)
			if err != nil {
				responseError(w, "unable to write file", err, http.StatusInternalServerError)
				return
			}
		}
		log.Println("Uploaded", fileName)
	}
	// set current parameters set
	_params = params
	w.WriteHeader(http.StatusOK)
	return
}

// ParamsHandler sets different options for the server
func ParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data, err := json.Marshal(_params)
		if err != nil {
			msg := "unable to marshal parameters"
			responseError(w, msg, err, http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	defer r.Body.Close()
	var params TFParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		msg := "unable to decode parameters"
		responseError(w, msg, err, http.StatusInternalServerError)
		return
	}
	log.Println("update TF parameters", params)
	if !strings.HasPrefix(params.Labels, "/") {
		params.Labels = fmt.Sprintf("%s/%s", _config.ModelDir, params.Labels)
	}
	if !strings.HasPrefix(params.Model, "/") {
		params.Model = fmt.Sprintf("%s/%s", _config.ModelDir, params.Model)
	}
	// set current parameters set
	_params = params
	w.WriteHeader(http.StatusOK)
}

// ModelsHandler returns a list of known models
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	models, err := TFModels()
	if err != nil {
		msg := fmt.Sprintf("Unable to get TF models")
		responseError(w, msg, err, http.StatusInternalServerError)
	}
	responseJSON(w, models)
}

// DefaultHandler authenticate incoming requests and route them to appropriate handler
func DefaultHandler(w http.ResponseWriter, r *http.Request) {
	var templates Templates
	tmplData := make(map[string]interface{})
	tmplData["Base"] = _config.Base
	tmplData["Content"] = fmt.Sprintf("Hello from TFaaS")
	tmplData["Version"] = info()
	tmplData["Models"], _ = TFModels()
	tmplData["ModelDir"] = _config.ModelDir
	main := templates.Main(_tmplDir, tmplData)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(_header + main + _footer))
}

// AuthHandler authenticate incoming requests and route them to appropriate handler
func AuthHandler(w http.ResponseWriter, r *http.Request) {
	// check if server started with hkey file (auth is required)
	if Auth == "true" {
		status := auth(r)
		if !status {
			msg := "not allowed access this resource"
			responseError(w, msg, nil, http.StatusForbidden)
			return
		}
	}
	// increment GET/POST counters
	if r.Method == "GET" {
		atomic.AddUint64(&TotalGetRequests, 1)
	}
	if r.Method == "POST" {
		atomic.AddUint64(&TotalPostRequests, 1)
	}
	path := r.URL.Path
	if _config.Base != "" {
		path = strings.Replace(path, _config.Base, "", 1)
	}
	arr := strings.Split(path, "/")
	path = arr[1]
	switch path {
	case "upload":
		UploadHandler(w, r)
	case "delete":
		DeleteHandler(w, r)
	case "data":
		DataHandler(w, r)
	case "json":
		PredictHandler(w, r)
	case "proto":
		PredictProtobufHandler(w, r)
	case "image":
		ImageHandler(w, r)
	case "params":
		ParamsHandler(w, r)
	case "models":
		ModelsHandler(w, r)
	case "status":
		StatusHandler(w, r)
	case "netron":
		NetronHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
}

// StatusHandler handlers Status requests
func StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		msg := "wrong HTTP method"
		responseError(w, msg, nil, http.StatusMethodNotAllowed)
		return
	}
	// get cpu and mem profiles
	m, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()
	l, _ := load.Avg()
	c, _ := cpu.Percent(time.Millisecond, true)

	tmplData := make(map[string]interface{})
	tmplData["NGo"] = runtime.NumGoroutine()
	virt := Memory{Total: m.Total, Free: m.Free, Used: m.Used, UsedPercent: m.UsedPercent}
	swap := Memory{Total: s.Total, Free: s.Free, Used: s.Used, UsedPercent: s.UsedPercent}
	tmplData["Memory"] = Mem{Virtual: virt, Swap: swap}
	tmplData["Load"] = l
	tmplData["CPU"] = c
	tmplData["Uptime"] = time.Since(Time0).Seconds()
	tmplData["getRequests"] = TotalGetRequests
	tmplData["postRequests"] = TotalPostRequests
	data, err := json.Marshal(tmplData)
	if err != nil {
		msg := "unabel to marshal data"
		responseError(w, msg, err, http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
	return
}

// NetronHandler provides hook to netron visualization library for graphs,
// see https://github.com/lutzroeder/Netron
func NetronHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		msg := "wrong HTTP method"
		responseError(w, msg, nil, http.StatusMethodNotAllowed)
		return
	}
	var endPoint string
	base := strings.TrimLeft(_config.Base, "/")
	for _, v := range strings.Split(r.URL.Path, "/") {
		if v == "" || v == base || v == "netron" {
			continue
		}
		endPoint = fmt.Sprintf("%s/%s", endPoint, v)
	}
	var ifile string
	endPoint = strings.TrimLeft(endPoint, "/")
	if endPoint == "" || endPoint == "netron" {
		ifile = fmt.Sprintf("%s/netron/%s", _config.StaticDir, "view-browser.html")
	} else {
		ifile = fmt.Sprintf("%s/netron/%s", _config.StaticDir, endPoint)
	}
	//     fmt.Println("ifile", ifile)
	page, err := ioutil.ReadFile(ifile)
	if err != nil {
		msg := fmt.Sprintf("unable to load %s", r.URL.Path)
		responseError(w, msg, err, http.StatusInternalServerError)
		return
	}
	w.Write(page)
}

// DELETE APIs

// DeleteHandler authenticate incoming requests and route them to appropriate handler
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		msg := "wrong HTTP method"
		responseError(w, msg, nil, http.StatusMethodNotAllowed)
		return
	}
	model := r.FormValue("model")
	files, err := ioutil.ReadDir(_config.ModelDir)
	if err != nil {
		responseError(w, fmt.Sprintf("unable to read: %s", _config.ModelDir), err, http.StatusInternalServerError)
		return
	}
	for _, f := range files {
		if f.Name() == model {
			path := fmt.Sprintf("%s/%s", _config.ModelDir, f.Name())
			err = os.RemoveAll(path)
			if err != nil {
				responseError(w, fmt.Sprintf("unable to remove: %s", path), err, http.StatusInternalServerError)
				return
			}
		}
	}
	_cache.remove(model)
	w.WriteHeader(http.StatusOK)
}
