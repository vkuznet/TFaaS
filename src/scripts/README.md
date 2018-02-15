### How to use curl with protobuf
To use curl tool with protobuf message we need to encode and decode them
before passing them to curl

First, let's create our message in text format (call it proto.msg):
```
key: "attribute1"
value: 1.1
key: "attribute2"
value: 2.2
```
which correspond our scheme defined in tfaas.proto file
```
syntax = "proto3";
package tfaaspb;

message Row {
    repeated string key = 1;
    repeated float value = 2;
}

message DataFrame {
    repeated Row row = 1;
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

# our scripts/request does the following:
# pdir=/path # points to path where we keep our proto file
# PROTO=tfaas.proto # our proto file name
# REQUEST=tfaaspb.Row # our protobuf package name (tfaaspb) dot message name (Row)
# RESPONSE=tfaaspb.Predictions
# scurl="curl -L -k --key $HOME/.globus/userkey.pem --cert $HOME/.globus/usercert.pem"
# cat $MSG | protoc -I$pdir --encode $REQUEST $PROTO | \
#    $scurl -sS -X POST --data-binary @- $URL | \
#    protoc -I$pdir --decode $RESPONSE $PROTO
```

### References

Protobuf documentation:
- https://developers.google.com/protocol-buffers/

How to encode/decode messages:
- https://stackoverflow.com/questions/18873924/what-does-the-protobuf-text-format-look-like

How to send messages with curl:
- http://xmeblog.blogspot.com/2013/12/sending-protobuf-serialized-data-using.html
- https://gist.github.com/alexeypegov/7887216
