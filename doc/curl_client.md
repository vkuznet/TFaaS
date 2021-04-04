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

```
# /bundle API can be used to upload ML bundle file to TFaaS
# let's say we produce ML model using Keras
import tensorflow as tf
...
model = tf.keras.Sequential()
model.add(tf.keras.layers.Dense(128, input_dim=idim, activation='relu', name="inputs"))
...
model.add(tf.keras.layers.Dense(1, activation='sigmoid'))

model.compile(loss='binary_crossentropy',
              optimizer=tf.keras.optimizers.Adam(lr=1e-3),
              metrics=[tf.keras.metrics.BinaryAccuracy(name='accuracy'), tf.keras.metrics.AUC(name='auc')])

# train the model
model.fit(X_train, Y_train, epochs=2, batch_size=128, validation_data=(X_val,Y_val))

# save our model into 'model' dir
tf.saved_model.save(model, 'model')

# the saved ML model will have the following content:
ls model
assets         saved_model.pb variables

# now we can create tar-ball and upload it to TFaaS
tar cfz model.tar.gz model
curl -X POST -H "Content-Encoding: gzip" \
             -H "content-type: application/octet-stream" \
             --data-binary @/tmp/models.tar.gz http://localhost:8083/bundle

# this model will be available as 'model' such that we can obtain predictions
# using our input.json file
cat input.json
{"keys": [...], "values": [...], "model":"model"}

# here is actual curl call to get predictions from /json end-point
curl -s -X POST -H "Content-type: application/json" \
    -d@/path/input.json http://localhost:8083/json
```
