package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"tfaaspb"
	"time"

	"github.com/golang/protobuf/proto"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"

	logs "github.com/sirupsen/logrus"
)

// helper function to provide response
func responseError(w http.ResponseWriter, msg string, err error, code int) {
	logs.WithFields(logs.Fields{
		"Message": msg,
		"Error":   err,
	}).Error(msg)
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
		responseError(w, fmt.Sprintf("wrong method: %v", r.Method), nil, http.StatusMethodNotAllowed)
		return
	}
	// Read image
	imageFile, header, err := r.FormFile("image")
	imageName := strings.Split(header.Filename, ".")
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
	session, err := tf.NewSession(_graph, _sessionOptions)
	if err != nil {
		responseError(w, "Unable to create new session", err, http.StatusInternalServerError)
		return
	}
	defer session.Close()
	if VERBOSE > 0 {
		devices, err := session.ListDevices()
		if err == nil {
			logs.WithFields(logs.Fields{
				"Devices": devices,
			}).Info("node availbility")
		} else {
			logs.WithFields(logs.Fields{
				"Error": err,
			}).Info("node availbility")
		}
	}
	output, err := session.Run(
		map[tf.Output]*tf.Tensor{
			_graph.Operation(_inputNode).Output(0): tensor,
		},
		[]tf.Output{
			_graph.Operation(_outputNode).Output(0),
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
	if len(_labels) < topN {
		topN = len(_labels)
	}
	responseJSON(w, ClassifyResult{
		Filename: "input", // TODO: we may replace the input name here to something meaningful
		Labels:   findBestLabels(probs, topN),
	})
}

// PredictProtobufHandler send prediction from TF ML model
func PredictProtobufHandler(w http.ResponseWriter, r *http.Request) {
	if !(r.Method == "POST") {
		logs.WithFields(logs.Fields{
			"Method": r.Method,
		}).Error("call PredictHandler with")
		w.WriteHeader(http.StatusMethodNotAllowed)
		responseError(w, fmt.Sprintf("wrong method: %v", r.Method), nil, http.StatusMethodNotAllowed)
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
		logs.WithFields(logs.Fields{
			"Data": recs,
		}).Info("received")
	}

	// example how to use tfaaspb protobuffer to ship back prediction data
	var objects []*tfaaspb.Class
	objects = append(objects, &tfaaspb.Class{Label: "higgs", Probability: float32(0.2)})
	objects = append(objects, &tfaaspb.Class{Label: "qcd", Probability: float32(0.8)})
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
		logs.WithFields(logs.Fields{
			"Method": r.Method,
		}).Error("call PredictHandler with")
		w.WriteHeader(http.StatusMethodNotAllowed)
		responseError(w, fmt.Sprintf("wrong method: %v", r.Method), nil, http.StatusMethodNotAllowed)
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
		logs.WithFields(logs.Fields{
			"Data": recs,
		}).Info("received")
	}

	// generate predictions
	probs, err := makePredictions(recs)
	if err != nil {
		responseError(w, "unable to make predictions", err, http.StatusInternalServerError)
		return
	}
	responseJSON(w, probs)
}

// POST methods

// UploadHandler uploads TF models into the server
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	if VERBOSE > 0 {
		logs.WithFields(logs.Fields{
			"Header": r.Header,
		}).Println("HEADER UploadHandler", r.Header)
	}
	modelFile, header, err := r.FormFile("model")
	if err != nil {
		responseError(w, "unable to read input request", err, http.StatusInternalServerError)
		return
	}
	defer modelFile.Close()

	// prepare file name to write to
	arr := strings.Split(header.Filename, "/")
	fname := arr[len(arr)-1]
	modelFileName := fmt.Sprintf("%s/%s", _config.ModelDir, fname)

	// read data from request and write it out to our local file
	data, err := ioutil.ReadAll(modelFile)
	if err != nil {
		responseError(w, "unable to read model file", err, http.StatusInternalServerError)
		return
	}
	err = ioutil.WriteFile(modelFileName, data, 0644)
	if err != nil {
		responseError(w, "unable to write model file", err, http.StatusInternalServerError)
		return
	}
	logs.WithFields(logs.Fields{
		"File": modelFileName,
	}).Info("Uploaded new model")
	w.WriteHeader(http.StatusOK)
	return
}

// ParamsHandler sets different options for the server
func ParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		out, err := json.Marshal(_config)
		if err != nil {
			responseError(w, "unable to marshal data", err, http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(out)
		return
	}
	defer r.Body.Close()
	var conf = Configuration{}
	err := json.NewDecoder(r.Body).Decode(&conf)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("ParamsHandler unable to marshal", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logs.WithFields(logs.Fields{
		"Set": conf.Params(),
	}).Info("update server settings")
	VERBOSE = conf.Verbose
	if conf.LogFormatter == "json" {
		logs.SetFormatter(&logs.JSONFormatter{})
	} else if conf.LogFormatter == "text" {
		logs.SetFormatter(&logs.TextFormatter{})
	} else {
		logs.SetFormatter(&logs.TextFormatter{})
	}
	if conf.InputNode != "" {
		_inputNode = conf.InputNode
	}
	if conf.OutputNode != "" {
		_outputNode = conf.OutputNode
	}
	if conf.ConfigProto != "" {
		_configProto = conf.ConfigProto
		_sessionOptions = readConfigProto(_configProto)
	}
	if conf.ModelLabels != "" {
		_modelLabels = conf.ModelLabels
	}
	if conf.ModelName != "" {
		_modelName = conf.ModelName
		if !strings.HasPrefix(_modelName, "/") {
			_modelName = fmt.Sprintf("%s/%s", _modelDir, _modelName)
		}
		err := loadModel(_modelName, _modelLabels)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Error":  err,
				"Model":  _modelName,
				"Labels": _modelLabels,
			}).Error("unable to open TF model")
		}
	}
	w.WriteHeader(http.StatusOK)
	return
}

// ModelsHandler authenticate incoming requests and route them to appropriate handler
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	files, err := ioutil.ReadDir(_modelDir)
	if err != nil {
		log.Fatal(err)
		responseError(w, fmt.Sprintf("unable to open: %s", _modelDir), err, http.StatusInternalServerError)
		return
	}
	responseJSON(w, files)
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
	if Auth == "true" {
		status := auth(r)
		if !status {
			msg := "You are not allowed to access this resource"
			http.Error(w, msg, http.StatusForbidden)
			return
		}
	}
	arr := strings.Split(r.URL.Path, "/")
	path := arr[len(arr)-1]
	switch path {
	case "upload":
		UploadHandler(w, r)
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
	default:
		DefaultHandler(w, r)
	}
}
