### Instructions
create proto file describing our input data, e.g.
```
syntax = "proto3";
package tfaaspb;

// Detector represents a CMS detector with name and x,y,z coordinates
message Detector {
    string name = 1;
    repeated float x = 2;
    repeated float y = 3;
    repeated float z = 4;
}

// Hits is a collection of detector elements
message Hits {
    repeated Detector det = 1;
}

// Row is a collection of keys and values
message Row {
    repeated string key = 1;
    repeated float value = 2;
    string model = 3;
}

// DataFrame is a collection of rows
message DataFrame {
    repeated Row row = 1;
}

// Class represents response from the server, it contains class label name and probability
message Class {
    string label = 1;
    float probability = 2;
}

// Predictions is collection of class probabilities
message Predictions {
    repeated Class prediction = 1;
}
```
Here input file represents hits from various detectors or rows (key-value
pair) of some dataframe, while output provides
description of out model output, i.e. class name and probability.
For format description see [protobuf](https://developers.google.com/protocol-buffers)
web site.

Download protoc (compiler) for your OS from
[protobuf](https://github.com/google/protobuf/releases) release download web site.
We need to pick up protoc-X.Y.Z-OS.zip file.

Generate protobuffer code to (de)-serialize our proto file
```
# generate Go code
protoc -I=$PWD/src/proto --go_out=$PWD/src/Go/tfaaspb $PWD/src/proto/tfaas.proto
# generate C++ code
protoc -I=$PWD/src/proto --cpp_out=$PWD/src/cpp $PWD/src/proto/tfaas.proto
# generate Python code
protoc -I=$PWD/src/proto --python_out=$PWD/src/python $PWD/src/proto/tfaas.proto
```

### How to generate TensorFlow proto config APIs
To generate TensorFlow proto config API we need to use the following commands:
```
# create local area to store TF APIs
mkdir tfconfig
# link to tensorflow source code
ln -s /path_to_tf/tensorflow tensorflow
# use protoc to generate TF proto config Go APIs
protoc -I=$PWD/tensorflow:$PWD/tensorflow/tensorflow/core/protobuf --go_out=$PWD/tfconfig $PWD/tensorflow/tensorflow/core/protobuf/config.proto
```
