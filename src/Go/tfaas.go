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

// ModelDir keeps location of TF models
var ModelDir string

// ModelName keeps name of TF model to use
var ModelName string

// ModelLabels keeps name of label file to use
var ModelLabels string

// ConfigProto keeps name of config proto to use
var ConfigProto string

// UploadResponse represents response from server to the client about upload of the model
type UploadResponse struct {
	Bytes int64  `json:"bytes"` // bytes of the uploaded model
	Hash  string `json:"hash"`  // hash of the uploaded model
}

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
	return fmt.Sprintf("<Params dir=%s model=%s labels=%s inputNode=%s outptuNode=%s configProt=%s verbose=%d log=%s>", c.ModelDir, c.ModelName, c.ModelLabels, c.InputNode, c.OutputNode, c.ConfigProto, c.Verbose, c.LogFormatter)
}

// global variables to hold TF graph and labels
var (
	_graph          *tf.Graph
	_labels         []string
	_sessionOptions *tf.SessionOptions
)

// global config
var _config Configuration

// global variable which we initialize once
var _userDNs []string

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
	_graph = tf.NewGraph()
	if err := _graph.Import(model, ""); err != nil {
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
		_labels = append(_labels, scanner.Text())
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
	session, err := tf.NewSession(_graph, _sessionOptions)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	results, err := session.Run(
		map[tf.Output]*tf.Tensor{_graph.Operation(InputNode).Output(0): tensor},
		[]tf.Output{_graph.Operation(OutputNode).Output(0)},
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
	session, err := tf.NewSession(graph, _sessionOptions)
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
		if i >= len(_labels) {
			break
		}
		resultLabels = append(resultLabels, LabelResult{Label: _labels[i], Probability: p})
	}
	// Sort by probability
	sort.Sort(ByProbability(resultLabels))
	// Return top N labels
	return resultLabels[:topN]
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
			_graph.Operation(InputNode).Output(0): tensor,
		},
		[]tf.Output{
			_graph.Operation(OutputNode).Output(0),
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

// SetHandler sets different options for the server
func SetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var conf = Configuration{}
	err := json.NewDecoder(r.Body).Decode(&conf)
	if err != nil {
		logs.WithFields(logs.Fields{
			"Error": err,
		}).Error("SetHandler unable to marshal", err)
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
		InputNode = conf.InputNode
	}
	if conf.OutputNode != "" {
		OutputNode = conf.OutputNode
	}
	if conf.ConfigProto != "" {
		ConfigProto = _config.ConfigProto
		_sessionOptions = readConfigProto(ConfigProto)
	}
	if ModelDir != "" {
		ModelDir = _config.ModelDir
	}
	if ModelName != "" {
		ModelName = _config.ModelName
	}
	if ModelLabels != "" {
		ModelLabels = _config.ModelLabels
	}
	w.WriteHeader(http.StatusOK)
	return
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
	case "set":
		SetHandler(w, r)
	default:
		DefaultHandler(w, r)
	}
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
	InputNode = _config.InputNode
	OutputNode = _config.OutputNode
	ConfigProto = _config.ConfigProto
	ModelDir = _config.ModelDir
	ModelName = _config.ModelName
	ModelLabels = _config.ModelLabels
	Auth = _config.Auth

	if ModelName != "" {
		err = loadModel(ModelName, ModelLabels)
		if err != nil {
			logs.WithFields(logs.Fields{
				"Error":  err,
				"Model":  ModelName,
				"Labels": ModelLabels,
			}).Error("unable to open TF model")
		}
		logs.WithFields(logs.Fields{
			"Auth":        Auth,
			"Model":       ModelName,
			"Labels":      ModelLabels,
			"InputNode":   InputNode,
			"OutputNode":  OutputNode,
			"ConfigProto": ConfigProto,
		}).Info("serving TF model")
	} else {
		logs.Warn("No model file is supplied, will unable to run inference")
	}

	http.Handle("/models/", http.StripPrefix("/models/", http.FileServer(http.Dir(ModelDir))))
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
