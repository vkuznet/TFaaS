### Client interface
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

