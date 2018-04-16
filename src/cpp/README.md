### TFaaS C++ code base
This area contains thin library (TFClient) which can be used to access
TFaaS server via protobuf protocol.

### Installation
To compile a library you need to obtain protobuf codebase, see
recent [release](https://github.com/google/protobuf/releases). Compile it
and make it available on your system. Then adjust path settings in
Makefile.mk to point to your protobuf path and issue `make` to compile
the library and simple demo client code, see
[main.cc](https://github.com/vkuznet/TFaaS/blob/master/src/cpp/main.cc).
