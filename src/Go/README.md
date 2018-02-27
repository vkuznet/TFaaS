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
go get github.com/vkuznet/x509proxy
go get github.com/golang/protobuf
go get github.com/sirupsen/logrus
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
# create models directory which we can server, e.g. mkdir -p $PWD/models
# ensure that we create labels.csv file
# place our TF model named mode.pb into models area
# find out input and output node TF node names
# run the server
./tfaas -dir $PWD/models 
   -serverCert /data/certs/hostcert.pem -serverKey /data/certs/hostkey.pem
   -modelLabels labels.csv -modelName models/model.pb
   -inputNode input_1_1 -outputNode output_node0
```
Here we supply the following list of parameters:
- server cert/key files to start-up HTTPs server
- modelLabels file which contains list of labels used by our TF model
- modelName file which contains full dump (including weights) of our TF model
- input/outputNode names used in our TF model

### How to get predictions from tfaas server
Once `tfaas` is up and running (default port is 8083) we can use it to server
prediction for our input data. Let's create a simple (curl-based) client:
```
#!/bin/bash

# construct our input vector
cat > /tmp/sb_input.json << EOF
{
    "keys":["a1","a2","a3","a4","a5","a6","a7","a8","a9"],
    "values":[1,1,1,1,1,1,1,1,1]
}
EOF

# host settings
headers="Content-type: application/json"
host=https://localhost:8083

# send request to host/json end-point
curl -L -k --key userkey.pem --cert usercert.pem -H "$headers" -d @/tmp/sb_input.json $host/json
```
Here, we created our input vector in JSON data-format
```
{
    "keys":["a1","a2","a3","a4","a5","a6","a7","a8","a9"],
    "values":[1,1,1,1,1,1,1,1,1]
}
```
which contains list of keys (in our example 9 keys, "a1" till "a9") and
corresponding values (in our case all ones, but in general it is your
input values in float data type).

Then, we invoke curl call (with necessary options) and pass it
to `$host/json` URI. The passed body is our JSON input which we load
from `/tmp/sb_input.json` file we created before.

### tfaas server APIs
The `tfaas` server provides several APIs:
```
# here we define scurl as a shortcut to
# curl -L -k --key ~/.globus/userkey.pem --cert ~/.globus/usercert.pem

# to list available models
scurl https://localhost:8083/models/

# to fetch concrete model file
scurl https://localhost:8083/models/tf.model1

# to increase verbosity level of the server
scurl -XPOST -d '{"level":1}' https://localhost:8083/verbose

# query prediction for our image (if we run TFaaS as image classifier)
scurl https://localhost:8083/image -F 'image=@/path/file.png'

# use JSON API to get prediction for our input data
scurl -XPOST -d '{"keys":["a","b"],"values":[1.1,2.0]}' https://localhost:8083/json

# use Protobuf API to get prediction for out input message (proto.msg)
# see scripts/README.md area for more details
scripts/request proto.msg https://localhost:8083/proto
```
For protobuf messages, please consult this
[page](https://github.com/vkuznet/TFaaS/blob/master/src/proto/README.md)

