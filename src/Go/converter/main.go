package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	tf "github.com/galeone/tensorflow/tensorflow/go"
)

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
		log.Println("unable to import graph model", err)
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

func main() {
	var modelFile string
	flag.StringVar(&modelFile, "modelFile", "", "input file name")
	var output string
	flag.StringVar(&output, "output", "", "output file name")
	var labelFile string
	flag.StringVar(&labelFile, "labelFile", "", "labelFile file name")
	flag.Parse()

	graph, _, err := loadModel(modelFile, labelFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	sGraph := ""
	for _, op := range graph.Operations() {
		fmt.Println(op.Name(), op.Type(), op.NumOutputs())
		for i := 0; i < op.NumOutputs(); i++ {
			for _, c := range op.Output(i).Consumers() {
				fmt.Println("consumer", c.Index)
			}
			if sGraph != "" {
				sGraph = fmt.Sprintf("%s -> %s(%s)", sGraph, op.Name(), op.Output(i).Shape())
			} else {
				sGraph = fmt.Sprintf("%s(%s)", op.Name(), op.Output(i).Shape())
			}
		}
	}
	fmt.Println("TF graph: ", sGraph)

	file, err := os.Create(output)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	nbytes, err := graph.WriteTo(writer)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("wrote", nbytes, "bytes")
}
