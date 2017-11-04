### Go server

This folder contains simple Go-based server which provides authentication
against CMS SiteDB and serve static files via HTTPs. To run it just do
```
go run tfaas.go
```
To access please use the following APIs:
```
# here we define scurl as a shortcut to
# curl -L -k --key ~/.globus/userkey.pem --cert ~/.globus/usercert.pem

# to list available models
scurl https://localhost:8083/models/

# to fetch concrete model file
scurl https://localhost:8083/models/tf.model1
```
