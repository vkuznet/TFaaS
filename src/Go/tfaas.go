package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"time"

	tf "github.com/galeone/tensorflow/tensorflow/go"
	"github.com/galeone/tensorflow/tensorflow/go/op"
	tg "github.com/galeone/tfgo"
)

// tfCache represent cache for TF 2.X models
var tfCache map[string]*tg.Model
var tfCacheParams map[string]TFParams

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
	Keys   []string  `json:"keys"`   // row attribute names
	Values []float32 `json:"values"` // row values
	Model  string    `json:"model"`  // TF model name to use
}

func (r *Row) String() string {
	return fmt.Sprintf("%v", r.Values)
}

// TFParams provides meta-data description of TF model to be used
type TFParams struct {
	Name        string   `json:"name"`        // model name
	Model       string   `json:"model"`       // model file name
	Labels      string   `json:"labels"`      // model labels file name
	Op          string   `json:"op"`          // model operation
	ImgChannels int64    `json:imgChannels`   // for img models number of img channels, color 3, black-white 1
	Options     []string `json:"options"`     // model options
	InputNode   string   `json:"inputNode"`   // model input node name
	OutputNode  string   `json:"outputNode"`  // model output node name
	Description string   `json:"description"` // model description
	TimeStamp   string   `json:"timestamp"`   // model timestamp
}

// String provides string representation of TFParams
func (p *TFParams) String() string {
	return fmt.Sprintf("<TFParams: name=%s model=%s description=%s labels=%s options=%v inputNode=%s outputNode=%s, timestamp=%s>", p.Name, p.Model, p.Description, p.Labels, p.Options, p.InputNode, p.OutputNode, p.TimeStamp)
}

// TFModel holds actual TF model (graph, labels, session options)
type TFModel struct {
	Params         TFParams
	Graph          *tf.Graph
	Labels         []string
	SessionOptions *tf.SessionOptions
}

// helper function to load TF graph and labels
func (m *TFModel) loadModel() error {
	if m.Graph != nil {
		return nil
	}
	modelPath := fmt.Sprintf("%s/%s/%s", _config.ModelDir, m.Params.Name, m.Params.Model)
	modelLabels := fmt.Sprintf("%s/%s/%s", _config.ModelDir, m.Params.Name, m.Params.Labels)
	if VERBOSE > 0 {
		log.Println("load to cache", modelPath, modelLabels)
	}
	graph, labels, err := loadModel(modelPath, modelLabels)
	if err != nil {
		return err
	}
	m.Graph = graph
	m.Labels = labels
	return nil
}

// TFCacheEntry holds all TFModels
type TFCacheEntry struct {
	TFModel TFModel
	Time    time.Time
}

// TFCache holds all TFModels
type TFCache struct {
	Models map[string]TFCacheEntry
	Limit  int
}

// add TFModel to the cache
func (c *TFCache) add(name string) error {
	if _, ok := c.Models[name]; ok {
		return nil
	}
	log.Println("load to cache", name)
	path := fmt.Sprintf("%s/%s", _config.ModelDir, name)
	fname := fmt.Sprintf("%s/params.json", path)
	if VERBOSE > 0 {
		log.Println("add to TFCache", fname)
	}
	file, err := os.Open(fname)
	defer file.Close()
	if err != nil {
		return err
	}
	var params TFParams
	if err := json.NewDecoder(file).Decode(&params); err != nil {
		return err
	}
	if params.TimeStamp == "" {
		params.TimeStamp = time.Now().String()
	}
	if VERBOSE > 0 {
		log.Println("add to TFCache", params)
	}
	tfm := TFModel{Params: params}
	err = tfm.loadModel()
	if err == nil {
		c.Models[params.Name] = TFCacheEntry{TFModel: tfm, Time: time.Now()}
	} else {
		log.Println("unable to load TF model", err)

	}
	if VERBOSE > 0 {
		log.Println("add to TFCache", c)
	}
	return err
}

// remove given model from the cache
func (c *TFCache) remove(name string) {
	delete(c.Models, name)
}

// return TFModel from the cache
func (c *TFCache) get(name string) (TFModel, error) {
	if entry, ok := c.Models[name]; ok {
		return entry.TFModel, nil
	}
	// our model is not available yet in cache
	// check cache size and clean it up if necessary
	if len(c.Models) >= c.Limit {
		var oldestName string
		oldestTime := time.Now()
		for name, entry := range c.Models {
			if entry.Time.Unix() < oldestTime.Unix() {
				oldestName = name
				oldestTime = entry.Time
			}
		}
		delete(c.Models, oldestName)
	}
	// add new model into cache
	err := c.add(name)
	if err != nil {
		return TFModel{}, err
	}
	// return model from the cache
	entry, _ := c.Models[name]
	return entry.TFModel, nil
}

// global variables
var (
	_cache          TFCache            // local cache for TFModels
	_params         TFParams           // current params set
	_sessionOptions *tf.SessionOptions // TF session options
	_configProto    string             // protobuf configuration
)

// helper function to read TF config proto message provided in input file
func readConfigProto(fname string) *tf.SessionOptions {
	session := tf.SessionOptions{}
	if fname != "" {
		body, err := ioutil.ReadFile(fname)
		if err == nil {
			session = tf.SessionOptions{Config: body}
		} else {
			log.Println("unable to read TF config proto file", err)
		}
	}
	return &session
}

// helper function to load TF model
func loadModel(fname, flabels string) (*tf.Graph, []string, error) {
	var labels []string
	graph := tf.NewGraph()
	// Load inception model
	model, err := ioutil.ReadFile(fname)
	if err != nil {
		return graph, labels, err
	}
	if err := graph.Import(model, ""); err != nil {
		log.Println("unable to import graph model", fname, err)
		return graph, labels, err
	}
	// Load labels
	labelsFile, err := os.Open(flabels)
	if err != nil {
		return graph, labels, err
	}
	defer labelsFile.Close()
	scanner := bufio.NewScanner(labelsFile)
	// Labels are separated by newlines
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return graph, labels, err
	}
	log.Println("load TF model", fname, flabels)
	return graph, labels, nil
}

// helper function to determine which model in our repository for given model name
func tfVersion(name string) (string, error) {
	// if model area has assets, variables and saved_model.pb
	// we will use TF 2.X approach based on tfgo
	path := fmt.Sprintf("%s/%s", _config.ModelDir, name)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}
	var fnames []string
	for _, file := range files {
		fnames = append(fnames, file.Name())
	}
	if InList("assets", fnames) && InList("variables", fnames) && InList("saved_model.pb", fnames) {
		return "tf2", nil
	}
	return "tf1", nil
}

// helper function to generate predictions based on given row values
// either TF 2.X models via tfgo or TF 1.X models via graph loading
func makePredictions(row *Row) ([]float32, error) {
	name := _params.Name
	if row.Model != "" {
		name = row.Model
	}
	tfModel, err := tfVersion(name)
	if err != nil {
		return []float32{}, err
	}
	if tfModel == "tf2" {
		return makePredictions2(row)
	}
	return makePredictions1(row)
}

// helper function to read tg.Model and its parameters
func getModel(name string) (*tg.Model, error) {
	if tfCache == nil {
		tfCache = make(map[string]*tg.Model)
	}
	var model *tg.Model
	var ok bool
	model, ok = tfCache[name]
	if !ok {
		path := fmt.Sprintf("%s/%s", _config.ModelDir, name)
		model = tg.LoadModel(path, []string{"serve"}, nil)
		tfCache[name] = model
	}
	return model, nil
}

// helper function to read model parameters
func getModelParams(name string) (TFParams, error) {
	if tfCacheParams == nil {
		tfCacheParams = make(map[string]TFParams)
	}
	params, ok := tfCacheParams[name]
	if !ok {
		fname := fmt.Sprintf("%s/%s/params.json", _config.ModelDir, name)
		file, err := os.Open(fname)
		if err != nil {
			return params, err
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			return params, err
		}

		err = json.Unmarshal(data, &params)
		if err != nil {
			return params, err
		}
		tfCacheParams[name] = params
	}
	return params, nil
}

// helper function to generate predictions based on given row values
// based on tfgo
func makePredictionsTensor(name string, tensor *tf.Tensor) ([]float32, error) {
	// our input is a tf Tensor

	// load TF model, saved as keras with the following dir structure
	// assets saved_model.pb variables

	// look-up model from out cache
	if tfCache == nil {
		tfCache = make(map[string]*tg.Model)
		tfCacheParams = make(map[string]TFParams)
	}

	// load model parameters
	model, err := getModel(name)
	if err != nil {
		return []float32{}, err
	}
	params, err := getModelParams(name)
	if err != nil {
		return []float32{}, err
	}
	if params.Op == "" {
		msg := fmt.Sprintf("Model params does not contain proper op parameter")
		return []float32{}, errors.New(msg)
	}
	log.Println("model op", params.Op, "tensor", tensor)

	results := model.Exec([]tf.Output{
		model.Op("StatefulPartitionedCall", 0),
	}, map[tf.Output]*tf.Tensor{
		model.Op(params.Op, 0): tensor,
	})
	probs := results[0]
	value := probs.Value() // returns [][]float32 vector
	vals := value.([][]float32)
	return vals[0], nil
}

// helper function to generate predictions based on given row values
// based on tfgo
func makePredictions2(row *Row) ([]float32, error) {
	// our input is a vector, we wrap it into matrix ([ [1,1,...], [], ...])
	matrix := [][]float32{row.Values}
	// create tensor vector for our computations
	tensor, err := tf.NewTensor(matrix)
	if err != nil {
		return nil, err
	}

	// load TF model, saved as keras with the following dir structure
	// assets saved_model.pb variables
	name := _params.Name
	if row.Model != "" {
		name = row.Model
	}
	// look-up model from out cache
	if tfCache == nil {
		tfCache = make(map[string]*tg.Model)
	}
	var model *tg.Model
	var ok bool
	model, ok = tfCache[name]
	if !ok {
		path := fmt.Sprintf("%s/%s", _config.ModelDir, name)
		model = tg.LoadModel(path, []string{"serve"}, nil)
		tfCache[name] = model
	}

	//     path := fmt.Sprintf("%s/%s", _config.ModelDir, name)
	//     model := tg.LoadModel(path, []string{"serve"}, nil)
	results := model.Exec([]tf.Output{
		model.Op("StatefulPartitionedCall", 0),
	}, map[tf.Output]*tf.Tensor{
		model.Op("serving_default_inputs_input", 0): tensor,
	})
	probs := results[0]
	value := probs.Value() // returns [][]float32 vector
	vals := value.([][]float32)
	return vals[0], nil
}

// helper function to generate predictions based on given row values
// based on TF 1.X models
// influenced by: https://pgaleone.eu/tensorflow/go/2017/05/29/understanding-tensorflow-using-go/
func makePredictions1(row *Row) ([]float32, error) {
	// our input is a vector, we wrap it into matrix ([ [1,1,...], [], ...])
	matrix := [][]float32{row.Values}
	// create tensor vector for our computations
	tensor, err := tf.NewTensor(matrix)
	if err != nil {
		return nil, err
	}

	// load TF model
	model := _params.Name
	if row.Model != "" {
		model = row.Model
	}
	tfm, err := _cache.get(model)
	if err != nil {
		log.Println("unable to get model from cache", model, err)
		return nil, err
	}

	// Run inference with existing graph which we get from loadModel call
	session, err := tf.NewSession(tfm.Graph, _sessionOptions)
	if err != nil {
		return nil, err
	}
	defer session.Close()
	results, err := session.Run(
		map[tf.Output]*tf.Tensor{tfm.Graph.Operation(tfm.Params.InputNode).Output(0): tensor},
		[]tf.Output{tfm.Graph.Operation(tfm.Params.OutputNode).Output(0)},
		nil)
	if err != nil {
		return nil, err
	}

	// our model probabilities
	probs := results[0].Value().([][]float32)[0]
	return probs, nil
}

// helper function to create Tensor image repreresentation
func makeTensorFromImage(imageBuffer *bytes.Buffer, imageFormat string, nChannels int64) (*tf.Tensor, error) {
	tensor, err := tf.NewTensor(imageBuffer.String())
	if err != nil {
		return nil, err
	}
	graph, input, output, err := makeTransformImageGraph(imageFormat, nChannels)
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
func makeTransformImageGraph(imageFormat string, nChannels int64) (graph *tf.Graph, input, output tf.Output, err error) {
	s := op.NewScope()
	input = op.Placeholder(s, tf.String)
	// Decode PNG or JPEG
	var decode tf.Output
	if imageFormat == "png" {
		decode = op.DecodePng(s, input, op.DecodePngChannels(nChannels))
	} else {
		decode = op.DecodeJpeg(s, input, op.DecodeJpegChannels(nChannels))
	}
	output = op.ExpandDims(s, op.Cast(s, decode, tf.Float), op.Const(s.SubScope("make_batch"), int32(0)))
	graph, err = s.Finalize()
	return graph, input, output, err
}

// ByProbability holds label results in terms of probability values
type ByProbability []LabelResult

func (a ByProbability) Len() int           { return len(a) }
func (a ByProbability) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByProbability) Less(i, j int) bool { return a[i].Probability > a[j].Probability }

func findBestLabels(labels []string, probabilities []float32, topN int) []LabelResult {
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
