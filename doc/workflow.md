### Introduction
Here we present end-to-end example of how to server TensorFlow model
(build and trained in Python) within Go-server. The example is based
on two blogs:
- [go-tensorflow](https://nilsmagnus.github.io/post/go-tensorflow/)
- [image-recognition-api-go-tensorflow](https://outcrawl.com/image-recognition-api-go-tensorflow/) 

I adapted it to MNIST dataset such that you can build your TensforFlow
model in python, save it and use in Go-server to recognize MNIST images.

### Setup
You need to have in place [MNIST datasets](http://yann.lecun.com/exdb/mnist/)
and you may use PNG images which can be found [here](https://github.com/myleott/mnist_png).

Setup your tensorflow library

Build your favorite ML model in python, the one I used is shown below
You may run it as simple as
```
python model.py --params=model.json
# here I used model.json to put store model parameters, e.g.
{
    "learning_rate": 0.01,
    "batch_size": 100,
    "n_hidden_1": 256,
    "n_hidden_2": 256,
    "n_classes": 10,
    "n_input": 784,
    "save_path": "/tmp/TensorFlow/model",
    "data_path": "/opt/data/mnist"
}
```
Once you run your python code it will save TensorFlow model into provided area
(in this case /tmp/TensorFlow/model). The Go server code expects model to be
in ./mnistmodel, therefore we make appropriate link:
```
ln -s /tmp/TensorFlow/model mnistmodel
```

Run `go run server.go` with Go server code shown below.

Query your server to recognize given image, e.g.
```
curl localhost:8080/recognize -F 'image=@./mnist_png/testing/9/104.png'
{"filename":"104.png","labels":[9]}
```
As you can see our model correctly recorgnized that provided image from folder
"9" was assigned label 9.

### Python model example

```
#!/usr/bin/env python

""" Neural Network.
A 2-Hidden Layers Fully Connected Neural Network (a.k.a Multilayer Perceptron)
implementation with TensorFlow. This example is using the MNIST database
of handwritten digits (http://yann.lecun.com/exdb/mnist/).
This example is using TensorFlow layers, see 'neural_network_raw' example for
a raw implementation with variables.
Links:
    [MNIST Dataset](http://yann.lecun.com/exdb/mnist/).
Author: Aymeric Damien
Project: https://github.com/aymericdamien/TensorFlow-Examples/
"""

from __future__ import print_function
import json
import argparse

# TensorFlow module
import tensorflow as tf
from tensorflow.examples.tutorials.mnist import input_data

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--params", action="store",
            dest="params", default="model.json", help="Input model parameters (default model.json)")
        self.parser.add_argument("--verbose", action="store",
            dest="verbose", default=0, help="verbosity level")

def train(params):
    "Train function defines our ML/DL model and work with our data"
    # extact given parameters
    data_path = params.get('data_path', '/opt/data/mnist')
    batch_size = params.get('batch_size', 256)
    fout = params.get('save_path', '')
    n_input = params.get('n_input', 784)
    n_classes = params.get('n_classes', 10)
    n_hidden_1 = params.get('n_hidden_1', 256)
    n_hidden_2 = params.get('n_hidden_2', 256)
    learning_rate = params.get('learning_rate', 0.01)
    display_step = 1

    mnist = input_data.read_data_sets(data_path, one_hot=True)
    xdf = mnist.train.images
    labels = mnist.train.labels

    # tf Graph input, use explicit name='input' for GOLANG
    x = tf.placeholder("float", [None, n_input], name='inputdata')
    y = tf.placeholder("float", [None, n_classes])

    # Create model
    def multilayer_perceptron(x, weights, biases):
        # Hidden layer with RELU activation
        layer_1 = tf.add(tf.matmul(x, weights['h1']), biases['b1'])
        layer_1 = tf.nn.relu(layer_1)
        # Hidden layer with RELU activation
        layer_2 = tf.add(tf.matmul(layer_1, weights['h2']), biases['b2'])
        layer_2 = tf.nn.relu(layer_2)
        # Output layer with linear activation
        out_layer = tf.matmul(layer_2, weights['out']) + biases['out']
        return out_layer

    # Store layers weight & bias
    weights = {
        'h1': tf.Variable(tf.random_normal([n_input, n_hidden_1])),
        'h2': tf.Variable(tf.random_normal([n_hidden_1, n_hidden_2])),
        'out': tf.Variable(tf.random_normal([n_hidden_2, n_classes]))
    }
    biases = {
        'b1': tf.Variable(tf.random_normal([n_hidden_1])),
        'b2': tf.Variable(tf.random_normal([n_hidden_2])),
        'out': tf.Variable(tf.random_normal([n_classes]))
    }

    # Construct model
    pred = multilayer_perceptron(x, weights, biases)

    # Define loss and optimizer
    cost = tf.reduce_mean(tf.nn.softmax_cross_entropy_with_logits(logits=pred, labels=y))
    optimizer = tf.train.AdamOptimizer(learning_rate=learning_rate).minimize(cost)

    # Initialize the variables (i.e. assign their default value)
    init = tf.global_variables_initializer()

    # Running first session
    print("Starting session...")
    with tf.Session() as sess:

        # Run the initializer
        sess.run(init)

        # Training cycle
        for epoch in range(3):
            avg_cost = 0.
            total_batch = int(mnist.train.num_examples/batch_size)
            # Loop over all batches
            for i in range(total_batch):
                batch_x, batch_y = mnist.train.next_batch(batch_size)
                # Run optimization op (backprop) and cost op (to get loss value)
                _, c = sess.run([optimizer, cost], feed_dict={x: batch_x,
                                                              y: batch_y})
                # Compute average loss
                avg_cost += c / total_batch
            # Display logs per epoch step
            if epoch % display_step == 0:
                print("Epoch:", '%04d' % (epoch+1), "cost=", \
                    "{:.9f}".format(avg_cost))
        print("First Optimization Finished!")

        # use expicit name='inference' for GOLANG
        infer = tf.argmax(pred, 1, name="inference")

        # Test model
        correct_prediction = tf.equal(infer, tf.argmax(y, 1))
        # Calculate accuracy
        accuracy = tf.reduce_mean(tf.cast(correct_prediction, "float"))
        print("Accuracy:", accuracy.eval({x: mnist.test.images, y: mnist.test.labels}))

        # Save model weights to disk
        if fout:
            builder = tf.saved_model.builder.SavedModelBuilder(fout)
            # GOLANG note that we must tag our model so that we can retrieve it at inference-time
            # for that purpose we use model tag
            builder.add_meta_graph_and_variables(sess, ["model"])
            builder.save()

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    params = json.load(open(opts.params))
    train(params)

if __name__ == '__main__':
    main()
```

### Go server example
```
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
)

var (
	model *tf.SavedModel
)

type Result struct {
	Filename string  `json:"filename"`
	Labels   []int64 `json:"labels"`
}

func main() {
	if err := loadModel(); err != nil {
		log.Fatal(err)
		return
	}

	r := httprouter.New()
	r.POST("/recognize", recognizeHandler)
	log.Fatal(http.ListenAndServe(":8080", r))
}

func loadModel() error {
	var err error
	model, err = tf.LoadSavedModel("mnistmodel", []string{"model"}, nil)
	if err != nil {
		return err
	}
	return nil
}

func typeof(v interface{}) string {
	return fmt.Sprintf("%T", v)
}

func inputTensor(imageBuffer *bytes.Buffer) (*tf.Tensor, error) {
	img, _, err := image.Decode(imageBuffer)
	if err != nil {
		return nil, err
	}
	var pix []uint8
	switch tim := img.(type) {
	case *image.RGBA:
		pix = tim.Pix
	case *image.NRGBA:
		pix = tim.Pix
	case *image.Gray:
		pix = tim.Pix
	default:
		fmt.Println(typeof(img))
		return nil, errors.New("unrecognized image")
	}
	size := len(pix)
	arr := make([]float32, size)
	for k, v := range pix {
		arr[k] = float32(v)
	}
	var imageData [][]float32
	imageData = append(imageData, arr)
	return tf.NewTensor(imageData)
}

func responseError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func recognizeHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Read image
	imageFile, header, err := r.FormFile("image")
	if err != nil {
		responseError(w, "Could not read image", http.StatusBadRequest)
		return
	}
	defer imageFile.Close()
	var imageBuffer bytes.Buffer
	// Copy image data to a buffer
	io.Copy(&imageBuffer, imageFile)

	// Make tensor
	tensor, err := inputTensor(&imageBuffer)
	if err != nil {
		responseError(w, "Invalid image", http.StatusBadRequest)
		return
	}
	// here we extract from model.Graph two operations:
	// inputdata is an input vector
	// inference is an output predictions
	// both tags should be defined within python code
	output, err := model.Session.Run(
		map[tf.Output]*tf.Tensor{
			model.Graph.Operation("inputdata").Output(0): tensor,
		},
		[]tf.Output{
			model.Graph.Operation("inference").Output(0),
		},
		nil,
	)

	if err != nil {
		fmt.Printf("Error running the session with input, err: %s\n", err.Error())
		return
	}

	res := output[0].Value()
	switch v := res.(type) {
	case []int64:
		responseJSON(w, Result{
			Filename: header.Filename,
			Labels:   v,
		})
	}
}
```
