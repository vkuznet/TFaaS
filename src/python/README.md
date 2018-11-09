### TFaaS client
We provide pure python
[client](https://github.com/vkuznet/TFaaS/blob/master/src/python/tfaas_client.py)
to perform all necessary action against TFaaS server. Here is short
description of available APIs:

```
# setup url to point to your TFaaS server
url=http://localhost:8083

# create upload json file, which should include
# fully qualified model file name
# fully qualified labels file name
# model name you want to assign to your model file
# fully qualified parameters json file name
# For example, here is a sample of upload json file
{
    "model": "/path/model_0228.pb",
    "labels": "/path/labels.txt",
    "name": "model_name",
    "params":"/path/params.json"
}

# upload given model to the server
tfaas_client.py --url=$url --upload=upload.json

# list existing models in TFaaS server
tfaas_client.py --url=$url --models

# delete given model in TFaaS server
tfaas_client.py --url=$url --delete=model_name

# prepare input json file for querying model predictions
# here is an example of such file
{"keys":["attribute1", "attribute2"], values: [1.0, -2.0]}

# get predictions from TFaaS server
tfaas_client.py --url=$url --predict=input.json

# get image predictions from TFaaS server
# here we refer to uploaded on TFaaS ImageModel model
tfaas_client.py --url=$url --image=/path/file.png --model=ImageModel
```
