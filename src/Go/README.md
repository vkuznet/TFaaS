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
   2>&1 1>& tfaas.log < /dev/null &

```
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

# use JSON API to get prediction for our input data
scurl -XPOST -d '{"keys":["a","b"],"values":[1.1,2.0]}' https://localhost:8083/json

# use Protobuf API to get prediction for out input message (proto.msg)
# see scripts/README.md area for more details
scripts/request proto.msg https://localhost:8083/proto
```
