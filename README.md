### TensorFlow as a Service (TFaaS)

[![Build Status](https://travis-ci.org/vkuznet/TFaaS.svg?branch=master)](https://travis-ci.org/vkuznet/TFaaS)
[![Go Report Card](https://goreportcard.com/badge/github.com/vkuznet/TFaaS)](https://goreportcard.com/report/github.com/vkuznet/TFaaS)
[![Tweet](https://img.shields.io/twitter/url/http/shields.io.svg?style=social)](https://twitter.com/intent/tweet?text=TensorFlow%20as%20a%20service%20&url=https://github.com/vkuznet/TFaaS&hashtags=tensorflow,go,python)

A general purpose framework (written in Go) to serve TensorFlow models.
It provides reach and flexible set of APIs to efficiently access your
favorite TF models via HTTP interface. The TFaaS supports JSON and ProtoBuffer
data-formats.

The following set of APIs is provided:
- */upload* to push your favorite TF model to TFaaS server
- */delete* to delete your TF model from TFaaS server
- */models* to view existing TF models on TFaaS server
- */json* and */proto* to serve TF models predictions in corresponding
  data-format

### TFaaS deployment
The most convenient way to install TFaaS server is using docker image
(default port of TFaaS is 8083):
```
docker run --rm -h `hostname -f` -p 8083:8083 -i -t veknet/tfaas
```

Otherwise, see [install instructions](https://github.com/vkuznet/TFaaS/blob/master/doc/INSTALL.md)
to build and deploy TFaaS from source code.

### TFaaS interface
Clients communicate with TFaaS via HTTP protocol. Here we show 3 client
workflows using Curl, Python and C++ clients.

To upload the TF model we prepare a parameters *params.json* file describing our model:
```
{
  "name": "ImageModel", "model": "tf_model.pb", "labels": "labels.txt",
  "inputNode": "dense_4_input", "outputNode": "output_node0"
}
```
It lists model name, an alias which we can use later for choosing a model 
during inference step, a model and labels file names, as well as input and output
node names of our models which you can get by inspecting your TF model.

The TF model, in this case named as ImageModel, will be registered in TFaaS
for further use.

#### Curl client
To upload our model we'll use curl client and provide model name, the
aforementioned *params.json* file, the TF model itself as well as our
label file:
```
curl -X POST http://localhost:8083/upload -F 'name=ImageModel'
-F 'params=@/path/params.json'
-F 'model=@/path/tf_model.pb' -F 'labels=@/path/labels.txt'
```
Once model is uploaded, we can query TFaaS and see what is available.
This can be done as following:
```
# query which TF models are available
curl http://localhost:8083/models

# it will return a JSON documents describing our models, e.g.
[{"name":"ImageModel","model":"tf_model.pb","labels":"labels.txt",
  "options":null,"inputNode":"dense_4_input","outputNode":"output_node0"}]
```
To get predictions we invoke curl call with new image file and specify our
model name to use for inference:
```
curl https://localhost:8083/image -F 'image=@/path/file.png' -F 'model=ImageModel'
```

#### Python client
For Python client example we'll use
[tfaas_client.py](https://github.com/vkuznet/TFaaS/blob/master/src/python/tfaas_client.py).
Similar to Curl client use case we need to upload our model to TFaaS server.
This can be done by creating *upload.json* file with our upload parameters:
```
{
  "model": "/path/tf_model.pb", "labels": "/path/labels.txt",
  "name": "myModel", "params":"/path/params.json"
}
```
It includes full path to our TF model, labels and parameters files as well as
name of our TF model. Now we can run our python client:
```
# define url for TFaaS
url=http://localhost:8083

# upload our model
tfaas_client.py --url=$url --upload=upload.json

# view registered models in TFaaS server
tfaas_client.py --url=$url --models
```
Finally, we can ask for predictions by preparing *input.json* file which
contains our keys (the list of names of our parameters), their values (the list
of numerical values) and model name we want to use for inference, e.g.:
```
{"keys":["attr1", "attr2", ...], "values":[1,2,...], "name":"myModel"}
```
We can place the following call to get our predictions:
```
tfaas_client.py --url=$url --predict=input.json
```

#### C++ client
Here we present only code how to make inference call to TFaaS server:
```
#include <iostream>
#include <vector>
#include <sstream>
#include “TFClient.h”                              // include TFClient header

// main function
int main() {
    std::vector<std::string> attrs;                // define vector of attributes
    std::vector<float> values;                     // define vector of values
    auto url = “http://localhost:8083/proto”;      // define your TFaaS URL
    auto model = “MyModel";                        // name your model

    // fill out our data
    for(int i=0; i<42; i++) {                      // the model I tested had 42 parameters
        values.push_back(i);                       // create your vector values
        std::ostringstream oss;
        oss << i;
        attrs.push_back(oss.str());                // create your vector headers
    }

    // make prediction call
    auto res = predict(url, model, attrs, values); // get predictions from TFaaS
    for(int i=0; i<res.prediction_size(); i++) {
        auto p = res.prediction(i);                // fetch and print model predictions
        std::cout << "class: " << p.label() << " probability: " << p.probability() << std::endl;
    }
}
```

### TFaaS benchmarks
Benchmark results on CentOS, 24 cores, 32GB of RAM serving DL NN with
42x128x128x128x64x64x1x1 architecture (JSON and ProtoBuffer formats show similar performance):
- 400 req/sec for 100 concurrent clients, 1000 requests in total
- 480 req/sec for 200 concurrent clients, 5000 requests in total
