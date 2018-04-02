### Introduction
This area contains CMSSW thin library (TFClient) which can be used to
send requests to TFaaS service.
The communication is done via
[protobuf](https://developers.google.com/protocol-buffers/) interface.  The
TFClient provides [predict](https://github.com/vkuznet/TFaaS/blob/master/src/cpp/TFClient/plugins/TFClient.cc#L108)
function to call TFaaS. It sends
POST request via [libcurl](https://curl.haxx.se/libcurl)
library to TFaaS server. The request contains a
[Hits](https://github.com/vkuznet/TFaaS/blob/master/src/proto/tfaas.proto)
data-structure and response has class predictions.
