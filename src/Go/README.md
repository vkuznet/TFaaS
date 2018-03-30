### Go server

This folder contains simple Go-based server which provides authentication
against CMS SiteDB and serve predictions for trained TF models via HTTPs. 

#### Installation
To install Go server you need to have `go` language on your system. If it is
not present please follow this simple instructions:
- download Go language for your favorite distribution, grab appropriate
tar-ball from [here](https://golang.org/dl/)
- setup `GOROOT` to point to isntalled go distribution, e.g.
`export GOROOT=/usr/local/go`, and setup `GOPATH` environment
to point to your local area where you go packages will be stored, e.g.
`export GOPATH=/path/gopath`
(please verify that `/path/gopath` directory exists and creates it if necessary)
- obtain GO tensorflow library and install it on your system, see
this [instructions](https://www.tensorflow.org/versions/master/install/install_go)
Once installed you'll need to get go tensorflow code. It will be compiled
against library you just downloaded and installed, please verify that
you'll setup proper `LD_LIBRARY_PATH` (on Linux) or `DYLD_LIBRARY_PATH` (on OSX).
To get go tensorflow librarys you just do
```
go get github.com/tensorflow/tensorflow/tensorflow/go
go get github.com/tensorflow/tensorflow/tensorflow/go/op
```
- download necessary dependencies for `tfaas`:
```
go get -u github.com/vkuznet/x509proxy
go get -u github.com/golang/protobuf/protoc-gen-go
go get -u github.com/sirupsen/logrus
```
- build `tfaas` Go server by running `make` and you'll get `tfaas` executable
  which is ready to server your models.

### Serving TF models with tfaas Go server
We need few pieces to run and serve TF models with our Go server. They are:
- server certificates to start HTTPs server
- TF model files (in protobuf data-format)
- input and output node names used in TF model
- predictions labels (optional)
We will outline each step in details below.

#### Generate self-signed host certificates
When you run HTTPs server you need to provide a host certificate to it.
You may generate self-signed certificates or obtain official ones from CA
authorities. Here we provide an example how to generate self-signed
certificates. To do that you need to have openssl library on your node
and execute the following command:
```
openssl req -new -newkey rsa:2048 -nodes -keyout server.key -out server.csr
```
Then, enter the following CSR details when prompted:
- Common Name: The FQDN (fully-qualified domain name) you want to secure with the certificate such as www.google.com, secure.website.org, *.domain.net, etc.
- Organization: The full legal name of your organization including the corporate identifier.
- Organization Unit (OU): Your department such as ‘Information Technology’ or ‘Website Security.’
- City or Locality: The locality or city where your organization is legally incorporated. Do not abbreviate.
- State or Province: The state or province where your organization is legally incorporated. Do not abbreviate.
- Country: The official two-letter country code (i.e. US, CH) where your organization is legally incorporated.

#### TF model files
To server TF model predictions we need to have valid TF model in protobuf
data-format (`.pb` extension). If you use TF code you can save your model
directly. For models created in using [Keras](https://keras.io/) library
you may use [keras-to-tensorflow](https://github.com/vkuznet/keras_to_tensorflow)
conversion tool which will convert your trained Keras model to TF one.
Please note, that even we require a binary version of TF model it is advised
to save the text version of TF model as well (usually `.pbtxt` extension).
The aforementioned tool will do that automatically. The text version of TF
model can be used to *discover* input and output node names used in your
TF layers.

#### TF input and output layers
Each TF model use input entry-point to feed your data (vector or matrix)
into TF model. This entry point has a name either assigned by TF code or
via explicit assignment. For example, if you create a Keras model
```
# here we declare Dense layer with output and input dimension
# and assign its name as my_dense_layer
Dense(out_dim, input_dim=input_dim, name="my_dense_layer")
```
or if you're using TF directly you can do it like this:
```
# here we create a placehoder for int32 input dimention
# and assign input_x as layer name
x = tf.placeholder(tf.int32, [None, input_dim], name="input_x")
```
You may inspect your TF model in text format to find out which layer
names where used, e.g.
```
cat model.pbtxt
node {
  name: "dense_10_input"
  op: "Placeholder"
  attr {
    key: "dtype"
    value {
      type: DT_FLOAT
    }
  }
  ...
node {
  name: "output_node0"
  op: "Identity"
  input: "dense_12/Sigmoid"
  attr {
    key: "T"
    value {
      type: DT_FLOAT
    }
  }
}
```
So, in this model the input layer name is `dense_10_input` and output layer
name is `output_node0`.

#### prediction labels
For models where there are multiple labels we need to create prediction labels
file. It is simple text file which lists its label on every line, e.g.
```
label1
label2
...
```

### How to run tfass server
Now we have all pieces to run `tfaas` server. We can do it as following:
```
# here is our configuration json file
cat config.json
{
    "port": 8083,
    "auth": "true",
    "modelDir": "models",
    "configProto": "",
    "base": "",
    "serverKey": "/opt/certs/server.key",
    "serverCrt": "/opt/certs/server.crt",
    "logFormatter": "text",
    "updateDNs": 600,
    "verbose": 0
}

# run the server with our config file
./tfaas -config config.json
```
If `tfaas` server quite and complained about CPU, e.g.
*Your CPU supports instructions that this TensorFlow binary was not compiled to use: SSE4.2 AVX AVX2 FMA*
it means that your TF library is not tuned (compiled) for your CPU. To resolve
the issue TF library needs to be complied from the source. For comprehensive
discussion please see this [post](https://stackoverflow.com/questions/47068709/your-cpu-supports-instructions-that-this-tensorflow-binary-was-not-compiled-to-u)
and this [one](https://stackoverflow.com/questions/41293077/how-to-compile-tensorflow-with-sse4-2-and-avx-instructions)

### tfaas server APIs
The `tfaas` server provides several APIs:
- GET APIs:
  - `/models` lists all available models/labels uploaded to TFaaS
  - `/params` lists model parameters to be used by TFaaS
  - `/models/<tf_model.pb>` fetches concrete model from TFaaS
- POST APIs:
  - `/upload` pushes your model to TFaaS
  - `/params` uploads new set of parameters to TFaaS
  - `/json` serves inference for given set of input parameters in JSON data-format
  - `/proto` serves inference in ProtoBuffer data-format
- DELETE APIs:
  - `/delete` deletes given model from TFaaS server

Here are few concrete examples of API usage:
```
# here we define scurl as a shortcut to
# curl -L -k --key ~/.globus/userkey.pem --cert ~/.globus/usercert.pem

# to list available models
scurl https://localhost:8083/models/

# to fetch concrete model file
scurl https://localhost:8083/models/tf.model1

# upload new model file to the server, it will be placed to modelDir area
# we must provide name, params, model and labels form values
scurl -i -X POST https://localhost:8083/upload -F 'name=luca' -F 'params=@/opt/cms/data/models/luca/params.json' -F 'model=@/opt/cms/data/models/luca/model_0228.pb' -F 'labels=@/opt/cms/data/models/luca/labels.csv'
# yet another example
scurl -i -X POST https://localhost:8083/upload -F 'name=image' -F 'params=@/opt/cms/data/models/higgs_qcd_muons/params.json' -F 'model=@/opt/cms/data/models/higgs_qcd_muons/tf_model_20180315.pb' -F 'labels=@/opt/cms/data/models/higgs_qcd_muons/labels.txt'

# once models are uploaded we can list them back via HTTP GET request
scurl https://localhost:8083/models/
[{"name":"image","model":"tf_model_20180315.pb","labels":"labels.txt","options":null,"inputNode":"input_1_1","outputNode":"output_node0"},{"name":"luca","model":"model_0228.pb","labels":"labels.csv","options":null,"inputNode":"dense_4_input","outputNode":"output_node0"}]

# we can remove given mode from TFaaS server
scurl -X DELETE https://localhost:8083/delete?model=image

# update server default TF parameters:
cat > params.json << EOF
{
    "name": "luca",
    "model": "model2.pb",
    "labels": "labels2.csv",
    "inputNode": "input_1_232323",
    "outputNode": "output_node_232323",
}
EOF
scurl -X POST -H "Content-type: application/json" -d @params.json https://localhost:8083/params

# get server parameters
scurl https://localhost:8083/params

# query prediction for our image (if we run TFaaS as image classifier)
scurl https://localhost:8083/image -F 'image=@/opt/cms/data/hep/train/RelValJpsiMuMu/run1_evt299719_lumi3.png' -F 'model=image'

# use JSON API to get prediction for our input data
scurl -XPOST -d '{"keys":["a","b"],"values":[1.1,2.0], "model":"luca"}' https://localhost:8083/json

# use Protobuf API to get prediction for out input message (proto.msg)
# see scripts/README.md area for more details

# here is an example of input.msg file
key: "attr1"
value: 1.1
key: "attr2"
value: 2.2
model: "image"

# now we can run inference as following
scripts/request $PWD/src/proto input.msg https://localhost:8083/proto
```
For protobuf messages, please consult this
[page](https://github.com/vkuznet/TFaaS/blob/master/src/proto/README.md)

