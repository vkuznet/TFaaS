### TFaaS C++ code base
This area contains thin library (TFClient) which can be used to access
TFaaS server via protobuf protocol.

### Installation
The TFaaS c++ code depends on:
- [google protobuf](https://github.com/google/protobuf/releases)
- [curl library](https://curl.haxx.se/download.html).
Please obtain and install them on your system.

Then adjust path settings in Makefile.mk to point to your protobuf path and
issue `make` to compile the library and simple demo client code, see
[main.cc](https://github.com/vkuznet/TFaaS/blob/master/src/cpp/main.cc).
