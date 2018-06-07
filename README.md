### TensorFlow as a Service (TFaaS)

A general purpose framework to serve TensorFlow models.
It provides reach and flexible set of APIs to efficiently manage your
favorite TF models. The TFaaS supports JSON and ProtoBuffer data-formats.

The following set of APIs is provided:
- */upload* to push your favorite TF model to TFaaS server
- */delete* to delete your TF model from TFaaS server
- */models* to view existing TF models on TFaaS server
- */json* and */proto* to serve TF models predictions in corresponding
  data-format

#### TFaaS deployment
Install TFaaS server via docker image:
```
docker run --rm -h `hostname -f` -p 8083:8083 -i -t veknet/tfaas
```

#### Clients
Clients communicate with TFaaS via HTTP protocol. Here we present 3 client
workflows: curl based, Python client and C++ client.

First, we need to prepare parameters to upload our model. We create
*params.json* file with the following content:
```
{
  "name": "ImageModel", "model": "tf_model.pb", "labels": "labels.txt",
  "inputNode": "dense_4_input", "outputNode": "output_node0"
}
It listss model name (an alias which we can use later for choosing a model 
during inference), a model and labels file names, as well as input and output
node names of our models which you can get by inspecting your TF model.
```

##### curl client
Upload your favorite model (we name it as *ImageModel*)
```
curl -X POST http://localhost:8083/upload -F 'name=ImageModel'
-F 'params=@/path/params.json'
-F 'model=@/path/tf_model.pb' -F 'labels=@/path/labels.txt'
```
Get predictions:
```
curl https://localhost:8083/image -F 'image=@/path/file.png' -F 'model=ImageModel'
```

##### python client
For python example we'll use
[tfaas_client.py](https://github.com/vkuznet/TFaaS/blob/master/src/python/tfaas_client.py).
First, we create *upload.json* file with our upload parameters:
```
{
  "model": "/path/tf_model.pb", "labels": "/path/labels.txt",
  "name": "myModel", "params":"/path/params.json"
}
```
Now we can upload and view this model as following:
```
# upload our model
tfaas_client.py --url=$url --upload=upload.json
# view our models
tfaas_client.py --url=$url --models
```
 
Finally, we can ask for prediction by preparing our *input.json* file
```
{"keys":["attr1", "attr2", ...], "values":[1,2,...], "name":"myModel"}
```
and look-up predictions for it:
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

#### TFaaS benchmarks
Benchmark results on CentOS, 24 cores, 32GB of RAM serving DL NN with
42x128x128x128x64x64x1x1 architecture:
- 400 req/sec for 100 concurrent clients, 1000 requests in total
- 480 req/sec for 200 concurrent clients, 5000 requests in total
JSON and ProtoBuffer formats show similar performance.
