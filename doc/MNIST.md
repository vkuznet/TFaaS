## End-to-end example using MNIST dataset

### Requirements (environment)
Here we provide full details to prepare used environment.
We will assume that you have a box where recent version of python is installed,
please note that instructions were tested with `Python 3.10.10`

```
# create mnist_env, here python refers to python 3.10.10
python -m venv mnist_env

# download mnist dataset for training purposes in numpy gziped arrays
curl -ksLO https://storage.googleapis.com/tensorflow/tf-keras-datasets/mnist.npz

# download MNIST dataset for training purposes in pkl.gz data-format
curl -ksLO https://s3.amazonaws.com/img-datasets/mnist.pkl.gz

# download MNIST images
# download MNIST actual images which we will use within inference
curl -O http://yann.lecun.com/exdb/mnist/train-images-idx3-ubyte.gz
```

### Train ML model
Below you can see fully tested Keras mased ML codebase to train
simple convolutional neural network over MNIST dataset:
```
#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : ktrain.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: Keras based ML network to train over MNIST dataset
"""

# system modules
import os
import sys
import json
import gzip
import pickle
import argparse

# third-party modules
import numpy as np
import tensorflow as tf
from tensorflow import keras
from tensorflow.keras import layers
from tensorflow.keras import backend as K
from tensorflow.python.tools import saved_model_utils


def modelGraph(model_dir):
    input_names = []
    output_names = []
    tag_sets = saved_model_utils.get_saved_model_tag_sets(model_dir)
#     print('The given SavedModel contains the following tag-sets:')
    for tag_set in sorted(tag_sets):
#         print("### tag_set", tag_set)
        print('%r' % ', '.join(sorted(tag_set)))
        meta_graph_def = saved_model_utils.get_meta_graph_def(model_dir, tag_set[0])
        for key in meta_graph_def.signature_def.keys():
            meta = meta_graph_def.signature_def[key]
            if hasattr(meta, 'inputs') and hasattr(meta, 'outputs'):
                inputs = meta.inputs
                outputs = meta.outputs
#                 print("### input", inputs, type(inputs), inputs.get('name'))
#                 print("### output", outputs, type(outputs), outputs.get('name'))
                input_signatures = list(meta.inputs.values())
                input_names = [signature.name for signature in input_signatures]
                if len(input_names) > 0:
                    output_signatures = list(meta.outputs.values())
                    output_names = [signature.name for signature in output_signatures]
    return input_names, output_names, meta_graph_def

def train(fin, fout=None, model_name=None, epoch=1, batch_size=128):
    """
    train function for MNIST
    """
    # Model / data parameters
    num_classes = 10
    input_shape = (28, 28, 1)

    # Load the data and split it between train and test sets
    # (x_train, y_train), (x_test, y_test) = keras.datasets.mnist.load_data()
    # or use input file
    f = gzip.open(fin, 'rb')
    if sys.version_info < (3,):
        mnist_data = pickle.load(f)
    else:
        mnist_data = pickle.load(f, encoding='bytes')
    f.close()
    (x_train, y_train), (x_test, y_test) = mnist_data

    # Scale images to the [0, 1] range
    x_train = x_train.astype("float32") / 255
    x_test = x_test.astype("float32") / 255
    # Make sure images have shape (28, 28, 1)
    x_train = np.expand_dims(x_train, -1)
    x_test = np.expand_dims(x_test, -1)
    print("x_train shape:", x_train.shape)
    print(x_train.shape[0], "train samples")
    print(x_test.shape[0], "test samples")


    # convert class vectors to binary class matrices
    y_train = keras.utils.to_categorical(y_train, num_classes)
    y_test = keras.utils.to_categorical(y_test, num_classes)

    # build model
    model = keras.Sequential(
        [
            keras.Input(shape=input_shape),
            layers.Conv2D(32, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Conv2D(64, kernel_size=(3, 3), activation="relu"),
            layers.MaxPooling2D(pool_size=(2, 2)),
            layers.Flatten(),
            layers.Dropout(0.5),
            layers.Dense(num_classes, activation="softmax"),
        ]
    )

    model.summary()
    print("model input", model.input, type(model.input), model.input.__dict__)
    print("model output", model.output, type(model.output), model.output.__dict__)

    # train model
    batch_size = batch_size
    epochs = epoch

    model.compile(loss="categorical_crossentropy", optimizer="adam", metrics=["accuracy"])
    model.fit(x_train, y_train, batch_size=batch_size, epochs=epochs, validation_split=0.1)

    # evaluate trained model
    score = model.evaluate(x_test, y_test, verbose=0)
    print("Test loss:", score[0])
    print("Test accuracy:", score[1])
    print("save model to", fout)
    if fout:
        model.save(fout)
        pbModel = '{}/saved_model.pb'.format(fout)
        pbtxtModel = '{}/saved_model.pbtxt'.format(fout)
        convert(pbModel, pbtxtModel)

        # get meta-data information about our ML model
        input_names, output_names, model_graph = modelGraph(model_name)
        print("### input", input_names)
        print("### output", output_names)
        # ML uses (28,28,1) shape, i.e. 28x28 black-white images
        # if we'll use color images we'll use shape (28, 28, 3)
        img_channels = input_shape[2]  # last item represent number of colors
        meta = {'name': model_name,
                'model': 'saved_model.pb',
                'labels': 'labels.txt',
                'img_channels': img_channels,
                'input_name': input_names[0].split(':')[0],
                'output_name': output_names[0].split(':')[0],
                'input_node': model.input.name,
                'output_node': model.output.name
        }
        with open(fout+'/params.json', 'w') as ostream:
            ostream.write(json.dumps(meta))
        with open(fout+'/labels.txt', 'w') as ostream:
            for i in range(0, 10):
                ostream.write(str(i)+'\n')
        with open(fout + '/model.graph', 'wb') as ostream:
            ostream.write(model_graph.SerializeToString())

def convert(fin, fout):
    """
    convert input model.pb into output model.pbtxt
    """
    import google.protobuf
    from tensorflow.core.protobuf import saved_model_pb2
    import tensorflow as tf

    saved_model = saved_model_pb2.SavedModel()

    with open(fin, 'rb') as f:
        saved_model.ParseFromString(f.read())

    with open(fout, 'w') as f:
        f.write(google.protobuf.text_format.MessageToString(saved_model))


class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input MNIST file")
        self.parser.add_argument("--fout", action="store",
            dest="fout", default="", help="Output models area")
        self.parser.add_argument("--model", action="store",
            dest="model", default="mnist", help="model name")
        self.parser.add_argument("--epoch", action="store",
            dest="epoch", default=1, help="number of epoch to train")
        self.parser.add_argument("--batch_size", action="store",
            dest="batch_size", default=128, help="batch size to use in training")

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    train(opts.fin, opts.fout,
          model_name=opts.model,
          epoch=opts.epoch,
          batch_size=opts.batch_size)

if __name__ == '__main__':
    main()
```

### Training process
We will train our model using the following command (for simplicity we skip
warning messages from TF and irrelevant printouts):
```
# here fout=mnist represents mnist directory where we'll stored our trained model
# and model=mnist is the name of the model we'll use later in inference
./ktrain.py --fin=./mnist.pkl.gz --fout=mnist --model=mnist
...
x_train shape: (60000, 28, 28, 1)
60000 train samples
10000 test samples
Model: "sequential"
_________________________________________________________________
 Layer (type)                Output Shape              Param #
=================================================================
 conv2d (Conv2D)             (None, 26, 26, 32)        320

 max_pooling2d (MaxPooling2D  (None, 13, 13, 32)       0
 )

 conv2d_1 (Conv2D)           (None, 11, 11, 64)        18496

 max_pooling2d_1 (MaxPooling  (None, 5, 5, 64)         0
 2D)

 flatten (Flatten)           (None, 1600)              0

 dropout (Dropout)           (None, 1600)              0

 dense (Dense)               (None, 10)                16010

=================================================================
Total params: 34,826
Trainable params: 34,826
Non-trainable params: 0
_________________________________________________________________

422/422 [==============================] - 37s 84ms/step - loss: 0.3645 - accuracy: 0.8898 - val_loss: 0.0825 - val_accuracy: 0.9772
Test loss: 0.09409885853528976
Test accuracy: 0.9703999757766724
save model to mnist

### input ['serving_default_input_1:0']
### output ['StatefulPartitionedCall:0']
```
When this process is over you'll find `mnist` directory with the following
content:
```
shell# ls mnist

assets            keras_metadata.pb model.graph       saved_model.pb    variables
fingerprint.pb    labels.txt        params.json       saved_model.pbtxt
```
- `saved_model.pb` represents trained ML model in protobuffer data-format
- `saved_model.pbtxt` represents trained ML model in text protobuffer representation
- `labels.txt` contains our image labels
- `params.json` contains meta-data used by TFaaS and it has the following content:
```
cat mnist/params.json | jq
{
  "name": "mnist",
  "model": "saved_model.pb",
  "labels": "labels.txt",
  "img_channels": 1,
  "input_name": "serving_default_input_1",
  "output_name": "StatefulPartitionedCall",
  "input_node": "input_1",
  "output_node": "dense/Softmax:0"
}
```
Here you see, that our ML model is called `mnist`, the model is stored in
`saved_model.pb` file, and more importantly this file contains the input and
output tensor names and nodes which we need to provide for TFaaS to server
our predictions.

### Inference server
Now, it is time to start our inference server. You can find its code in `src/go` area.
To build the code you need
```
# download TF library and includes for your OS, e.g. macOS build
curl -ksLO https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-darwin-x86_64-2.11.0.tar.gz

# provide TF include area location to go build command
# the /opt/tensorflow/include is are where TF includes are
export CGO_CPPFLAGS="-I/opt/tensorflow/include"

# compile the code
make

# it will produce tfaas executable

# to run the code we need to setup `DYLD_LIBRARY_PATH`
export DYLD_LIBRARY_PATH=/opt/tensorflow/lib
./tfaas -config config.json
```
where `config.json` has the following form (please refer for more details):
```
{
    "port": 8083,
    "modelDir": "models",
    "staticDir": "static",
    "configProto": "",
    "base": "",
    "serverKey": "",
    "serverCrt": "",
    "verbose": 1
}
```

Finally, we are ready for the inference part.
- upload your ML model to TFaaS server
```
# create tarball of your mnist ML trained model
tar cfz mnist.tar.gz mnist

# upload tarball to TFaaS server
curl -v -X POST -H "Content-Encoding: gzip" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @./mnist.tar.gz \
    http://localhost:8083/upload

# check your model presence
curl http://localhost:8083/models

# generate image from MNIST dataset you want to use for prediction
# img1.png will contain number 1, img4.png will contain number 4
./mnist_img.py --fout img1.png --imgid=3
./mnist_img.py --fout img4.png --imgid=2

# ask for prediction of your image
curl http://localhost:8083/predict/image -F 'image=@./img1.png' -F 'model=mnist'
[0,1,0,0,0,0,0,0,0,0]

curl http://localhost:8083/predict/image -F 'image=@./img4.png' -F 'model=mnist'
[0,0,0,0,1,0,0,0,0,0]
```
