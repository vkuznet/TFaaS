package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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
	model := r.FormValue("model")
	if model == "" {
		msg := fmt.Sprintf("unable to read %s model", model)
		responseError(w, msg, nil, http.StatusInternalServerError)
		return
	}
	// read image model
	params := TFParams{Name: model}
	tfm, err := loadTFModel(params)
	if err != nil {
		responseError(w, "unable to read image model", err, http.StatusInternalServerError)
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
	session, err := tf.NewSession(tfm.Graph, _sessionOptions)
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
		Filename: "input", // TODO: we may replace the input name here to something meaningful
		Labels:   findBestLabels(tfm.Labels, probs, topN),
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
		logs.WithFields(logs.Fields{
			"Inputs": records,
			"Probs":  probs,
		}).Info("respose")
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
		}).Info("UploadHandler")
	}
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
			logs.WithFields(logs.Fields{
				"FileName": header.Filename,
			}).Info(msg)
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
			if mkey != params.Name {
				msg := fmt.Sprintf("mismatch of mkey=%s and TFParam.Name=%s", mkey, params.Name)
				responseError(w, msg, err, http.StatusInternalServerError)
				return
			}
		}

		// write out content to our store
		err = ioutil.WriteFile(fileName, data, 0644)
		if err != nil {
			responseError(w, "unable to write file", err, http.StatusInternalServerError)
			return
		}
		logs.WithFields(logs.Fields{
			"File": fileName,
		}).Info("Uploaded")
	}
	// load model for our TF params
	_, err := loadTFModel(params)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error":  err,
			"Params": params.String(),
		}).Error("unable to open TF model")
	}
	_params = params // set current params set

	w.WriteHeader(http.StatusOK)
	return
}

// ParamsHandler sets different options for the server
func ParamsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data, err := json.Marshal(_params)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Error": err,
			}).Error("ParamsHandler unable to marshal")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(data)
		return
	}
	defer r.Body.Close()
	var params TFParams
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("ParamsHandler unable to marshal")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	logs.WithFields(logs.Fields{
		"Params": params,
	}).Info("update TF parameters")
	if !strings.HasPrefix(params.Labels, "/") {
		params.Labels = fmt.Sprintf("%s/%s", _config.ModelDir, params.Labels)
	}
	if !strings.HasPrefix(params.Model, "/") {
		params.Model = fmt.Sprintf("%s/%s", _config.ModelDir, params.Model)
	}
	// load model for our TF params
	_, err = loadTFModel(params)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error":  err,
			"Params": params.String(),
		}).Error("unable to open TF model")
	}
	_params = params // set current params set
	w.WriteHeader(http.StatusOK)
	return
}

// ModelsHandler authenticate incoming requests and route them to appropriate handler
func ModelsHandler(w http.ResponseWriter, r *http.Request) {
	var models []TFParams
	for _, tfm := range _models {
		models = append(models, tfm.Params)
	}
	responseJSON(w, models)
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
	default:
		DefaultHandler(w, r)
	}
}

// DELETE APIs
// DeleteHandler authenticate incoming requests and route them to appropriate handler
func DeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
	delete(_models, model)
	w.WriteHeader(http.StatusOK)
}
