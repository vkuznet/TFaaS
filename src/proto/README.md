### Instructions
create proto file describing our input data, e.g.
```
syntax = "proto3";
package tfaaspb;

message Detector {
    string name = 1;
    repeated float x = 2;
    repeated float y = 3;
    repeated float z = 4;
}

message Hits {
    repeated Detector det = 1;
}

message Row {
    repeated string k = 1;
    repeated float v = 2;
}

message DataFrame {
    repeated Row row = 1;
}

message Class {
    string name = 1;
    repeated float p = 2;
}

message Predictions {
    repeated Class cls = 1;
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
protoc/bin/protoc -I=$PWD/src/proto --go_out=$PWD/src/Go/tfaaspb $PWD/src/proto/tfaas.proto
# generate C++ code
protoc/bin/protoc -I=$PWD/src/proto --cpp_out=$PWD/src/cpp $PWD/src/proto/tfaas.proto
```
