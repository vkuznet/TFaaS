### TFaaS client
We provide pure python
[client](https://github.com/vkuznet/TFaaS/blob/master/src/python/tfaas_client.py)
to perform all necessary action against TFaaS server. Here is short
description of available APIs:

```
# setup url to point to your TFaaS server
url=http://localhost:8083

# create upload json file, which should include
# - fully qualified model file name
# - fully qualified labels file name
# - model name you want to assign to your model file
# - fully qualified model parameters json file name
# For example, here is a sample of upload json file
{
    "model": "/path/model.pb",
    "labels": "/path/labels.txt",
    "name": "model_name",
    "params":"/path/params.json"
}

# The model parameters json file is used on a TFaaS server side. It context
# should be the following:
# - model name (how your model will be named in TFaaS server,
#               usually match the one from upload.json file)
# - model file name (name of your model file will be used in TFaaS server,
#                    usually match the one from upload.json file)
# - model labels file name (similar to model file name but used for labels file)
# - description string (provide details about your model)
# - inputNode of your TF model (can be found by inspecting pbtxt)
# - outputNode of your TF model (can be found by inspecting pbtxt)
# For example:
{
    "name": "model_name",
    "model": "model.pb",
    "description": "my model description",
    "labels": "labels.txt",
    "inputNode": "dense_1_input",
    "outputNode": "output_node0"
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
