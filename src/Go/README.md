### Go server

This folder contains simple Go-based server which provides authentication
against CMS SiteDB and serve static files via HTTPs. To build the server
just use make
```
make
```
Here is an example how to run the server
```
nohup ./tfaas -dir $PWD/models 
   -serverCert /data/certs/hostcert.pem -serverKey /data/certs/hostkey.pem
   -modelLabels labels.csv -modelName models/model.pb
   -inputNode input_1_1 -outputNode output_node0
   2>&1 1>& tfaas.log < /dev/null &

```
Here we supply the following list of parameters:
- server cert/key files to start-up HTTPs server
- modelLabels file which contains list of labels used by our TF model
- modelName file which contains full dump (including weights) of our TF model
- input/outputNode names used in our TF model

To access please use the following APIs:
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

### Generate self-signed host certificates
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
