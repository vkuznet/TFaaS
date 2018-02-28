package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"sort"
	"strings"
	"tfaaspb"
	"time"

	"github.com/golang/protobuf/proto"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"

	logs "github.com/sirupsen/logrus"
	"github.com/vkuznet/x509proxy"
)

// VERBOSE controls verbosity of the server
var VERBOSE int

// InputNode represents input node name in TF graph
var InputNode string

// Auth represents flag to use authentication or not
var Auth string

// OutputNode represents input node name in TF graph
var OutputNode string

// ClassifyResult structure represents result of our TF model classification
type ClassifyResult struct {
	Filename string        `json:"filename"`
	Labels   []LabelResult `json:"labels"`
}

// LabelResult structure represents single result of TF model classification
type LabelResult struct {
	Label       string  `json:"label"`
	Probability float32 `json:"probability"`
}

// Row structure represents input set of attributes client will send to the server
type Row struct {
	Keys   []string  `json:"keys"`
	Values []float32 `json:"values"`
}

func (r *Row) String() string {
	return fmt.Sprintf("%v", r.Values)
}

// global variables to hold TF graph and labels
var (
	graph          *tf.Graph
	labels         []string
	sessionOptions *tf.SessionOptions
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
		if Auth == "true" {
			logs.Fatal("Neither proxy or user certs are found, use X509_USER_PROXY/X509_USER_KEY/X509_USER_CERT to set them up")
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

// helper function to read TF config proto message provided in input file
func readConfigProto(fname string) *tf.SessionOptions {
	session := tf.SessionOptions{}
	if fname != "" {
		body, err := ioutil.ReadFile(fname)
		if err == nil {
			session = tf.SessionOptions{Config: body}
		} else {
			logs.WithFields(logs.Fields{
				"Error": err,
			}).Error("Unable to read TF config proto file")
		}
	}
	return &session
}

// helper function to load TF model
func loadModel(fname, flabels string) error {
	// Load inception model
	model, err := ioutil.ReadFile(fname)
	if err != nil {
		return err
	}
	graph = tf.NewGraph()
	if err := graph.Import(model, ""); err != nil {
		return err
	}
	// Load labels
	labelsFile, err := os.Open(flabels)
	if err != nil {
		return err
	}
	defer labelsFile.Close()
	scanner := bufio.NewScanner(labelsFile)
	// Labels are separated by newlines
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// helper function to generate predictions based on given row values
// influenced by: https://pgaleone.eu/tensorflow/go/2017/05/29/understanding-tensorflow-using-go/
func makePredictions(row *Row) ([]float32, error) {
	// our input is a vector, we wrap it into matrix ([ [1,1,...], [], ...])
	matrix := [][]float32{row.Values}
	// create tensor vector for our computations
	tensor, err := tf.NewTensor(matrix)
	//tensor, err := tf.NewTensor(row.Values)
	if err != nil {
		return nil, err
	}

	// Run inference with existing graph which we get from loadModel call
	session, err := tf.NewSession(graph, sessionOptions)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	results, err := session.Run(
		map[tf.Output]*tf.Tensor{graph.Operation(InputNode).Output(0): tensor},
		[]tf.Output{graph.Operation(OutputNode).Output(0)},
		nil)
	if err != nil {
		return nil, err
	}

	// our model probabilities
	probs := results[0].Value().([][]float32)[0]
	return probs, nil
}

// helper function to create Tensor image repreresentation
func makeTensorFromImage(imageBuffer *bytes.Buffer, imageFormat string) (*tf.Tensor, error) {
	tensor, err := tf.NewTensor(imageBuffer.String())
	if err != nil {
		return nil, err
	}
	graph, input, output, err := makeTransformImageGraph(imageFormat)
	if err != nil {
		return nil, err
	}
	session, err := tf.NewSession(graph, sessionOptions)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	normalized, err := session.Run(
		map[tf.Output]*tf.Tensor{input: tensor},
		[]tf.Output{output},
		nil)
	if err != nil {
		return nil, err
	}
	return normalized[0], nil
}

// Creates a graph to decode an image
func makeTransformImageGraph(imageFormat string) (graph *tf.Graph, input, output tf.Output, err error) {
	s := op.NewScope()
	input = op.Placeholder(s, tf.String)
	// Decode PNG or JPEG
	var decode tf.Output
	if imageFormat == "png" {
		decode = op.DecodePng(s, input, op.DecodePngChannels(3))
	} else {
		decode = op.DecodeJpeg(s, input, op.DecodeJpegChannels(3))
	}
	output = op.ExpandDims(s, op.Cast(s, decode, tf.Float), op.Const(s.SubScope("make_batch"), int32(0)))
	graph, err = s.Finalize()
	return graph, input, output, err
}

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

type ByProbability []LabelResult

func (a ByProbability) Len() int           { return len(a) }
func (a ByProbability) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProbability) Less(i, j int) bool { return a[i].Probability > a[j].Probability }

func findBestLabels(probabilities []float32, topN int) []LabelResult {
	// Make a list of label/probability pairs
	var resultLabels []LabelResult
	for i, p := range probabilities {
		if i >= len(labels) {
			break
		}
		resultLabels = append(resultLabels, LabelResult{Label: labels[i], Probability: p})
	}
	// Sort by probability
	sort.Sort(ByProbability(resultLabels))
	// Return top N labels
	return resultLabels[:topN]
}

//
// HTTP handlers
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
	session, err := tf.NewSession(graph, sessionOptions)
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
			graph.Operation(InputNode).Output(0): tensor,
		},
		[]tf.Output{
			graph.Operation(OutputNode).Output(0),
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
	if len(labels) < topN {
		topN = len(labels)
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

// helper data structure to change verbosity level of the running server
type level struct {
	Level int `json:"level"`
}

// VerboseHandler sets verbosity level for the server
func VerboseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logs.Warn("Unable to parse request body: ", err)
	}
	var v level
	err = json.Unmarshal(body, &v)
	if err == nil {
		logs.Info("Switch to verbose level: ", v.Level)
		VERBOSE = v.Level
	}
	w.WriteHeader(http.StatusOK)
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
	case "data":
		DataHandler(w, r)
	case "json":
		PredictHandler(w, r)
	case "proto":
		PredictProtobufHandler(w, r)
	case "image":
		ImageHandler(w, r)
	case "verbose":
		VerboseHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
}

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "models", "local directory to serve by this server")
	var port int
	flag.IntVar(&port, "port", 8083, "server port")
	flag.StringVar(&Auth, "auth", "true", "Use authentication or not")
	var serverKey string
	flag.StringVar(&serverKey, "serverKey", "", "server key file")
	var serverCert string
	flag.StringVar(&serverCert, "serverCert", "", "server cert file")
	var modelName string
	flag.StringVar(&modelName, "modelName", "", "model protobuf file name")
	var modelLabels string
	flag.StringVar(&modelLabels, "modelLabels", "", "model labels")
	flag.StringVar(&InputNode, "inputNode", "", "TF input node name")
	flag.StringVar(&OutputNode, "outputNode", "", "TF output node name")
	var configProto string
	flag.StringVar(&configProto, "configProto", "", "TF proto config file")
	flag.Parse()

	var err error

	// create session options from given config TF proto file
	sessionOptions = readConfigProto(configProto)

	if modelName != "" {
		err = loadModel(modelName, modelLabels)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Error":  err,
				"Model":  modelName,
				"Labels": modelLabels,
			}).Error("unable to open TF model")
		}
		logs.WithFields(logs.Fields{
			"Auth":        Auth,
			"Model":       modelName,
			"Labels":      modelLabels,
			"InputNode":   InputNode,
			"OutputNode":  OutputNode,
			"ConfigProto": configProto,
		}).Info("serving TF model")
	} else {
		logs.Warn("No model file is supplied, will unable to run inference")
	}

	http.Handle("/models/", http.StripPrefix("/models/", http.FileServer(http.Dir(dir))))
	http.HandleFunc("/", AuthHandler)
	addr := fmt.Sprintf(":%d", port)
	if serverKey != "" && serverCert != "" {
		server := &http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				ClientAuth: tls.RequestClientCert,
			},
		}
		if _, err := os.Open(serverKey); err != nil {
			logs.WithFields(logs.Fields{
				"Error": err,
				"File":  serverKey,
			}).Error("unable to open server key file")
		}
		if _, err := os.Open(serverCert); err != nil {
			logs.WithFields(logs.Fields{
				"Error": err,
				"File":  serverCert,
			}).Error("unable to open server cert file")
		}
		logs.WithFields(logs.Fields{"Addr": addr}).Info("Starting HTTPs server")
		err = server.ListenAndServeTLS(serverCert, serverKey)
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
