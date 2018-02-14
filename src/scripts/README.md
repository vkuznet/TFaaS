### How to use curl with protobuf
To use curl tool with protobuf message we need to encode and decode them
before passing them to curl

First, let's create our message in text format (call it proto.msg):
```
pair {
    key: "attribute1"
    value: 1.1
}
pair {
    key: "attribute2"
    value: 2.2
}
```
which correspond our scheme defined in tfaas.proto file
```
syntax = "proto3";
package tfaaspb;

message Pair {
    string key = 1;
    float value = 2;
}

message DataFrame {
    repeated Pair pair = 1;
}
```


To encode the message we use the following command:
```
# here we define path with -I option where our protobuf file description (tfaas.proto) reside
cat proto.msg | protoc -I/Users/vk/CMS/DMWM/GIT/TFaaS/src/proto --encode=tfaaspb.DataFrame tfaas.proto > row.bin
```
to decode the message we use:
```
protoc -I/Users/vk/CMS/DMWM/GIT/TFaaS/src/proto --decode=tfaaspb.DataFrame tfaas.proto < row.bin
```
Please note that we use full namespace `tfaaspb.DataFrame` as defined in our schema.

Then we can extend this example to send our message to our server using script/request:
```
scripts/request proto.msg https://localhost:8083/predictproto
```

### References

Protobuf documentation:
https://developers.google.com/protocol-buffers/

How to encode/decode messages:
https://stackoverflow.com/questions/18873924/what-does-the-protobuf-text-format-look-like

How to send messages with curl:
http://xmeblog.blogspot.com/2013/12/sending-protobuf-serialized-data-using.html
http://xmeblog.blogspot.com/2013/12/sending-protobuf-serialized-data-using.html
