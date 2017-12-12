### Go server

This folder contains simple Go-based server which provides authentication
against CMS SiteDB and serve static files via HTTPs. To build the server
just use make
```
make
```
Here is an example how to run the server
```
./tfaas -dir $PWD/modles -serverKey server.key -serverCert server.crt
```
To access please use the following APIs:
```
# here we define scurl as a shortcut to
# curl -L -k --key ~/.globus/userkey.pem --cert ~/.globus/usercert.pem

# to list available models
scurl https://localhost:8083/models/

# to fetch concrete model file
scurl https://localhost:8083/models/tf.model1

# to get protobuffer message
scurl https://localhost:8083/predict
```
