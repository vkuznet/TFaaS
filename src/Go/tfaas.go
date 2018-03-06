package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"

	logs "github.com/sirupsen/logrus"
)

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

// global variables
var (
	_graph                                                                     *tf.Graph
	_labels                                                                    []string
	_sessionOptions                                                            *tf.SessionOptions
	_config                                                                    Configuration
	_inputNode, _outputNode, _modelDir, _modelName, _modelLabels, _configProto string
)

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
	logs.WithFields(logs.Fields{
		"Model":  _modelName,
		"Labels": _modelLabels,
	}).Info("load TF model")
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
		map[tf.Output]*tf.Tensor{_graph.Operation(_inputNode).Output(0): tensor},
		[]tf.Output{_graph.Operation(_outputNode).Output(0)},
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
