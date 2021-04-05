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

#### recipe on lxplus
```
# download sources
curl -ksLO https://github.com/protocolbuffers/protobuf/releases/download/v3.15.7/protobuf-cpp-3.15.7.tar.gz
curl -ksLO https://github.com/protocolbuffers/protobuf/releases/download/v3.15.7/protoc-3.15.7-linux-x86_64.zip

# create dir structur
mkdir protoc
cd protoc
unzip protoc-3.15.7-linux-x86_64.zip

cd ../protobuf-cpp-3.15.7
./configure --prefix=$PWD/install
make -j 15
make install

# setup environment
export PATH=/afs/cern.ch/user/v/valya/workspace/protoc/bin:$PATH
export LD_LIBRARY_PATH=/afs/cern.ch/user/v/valya/workspace/protobuf-3.15.7/install/lib:$LD_LIBRARY_PATH
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# go to TFaaS
cd ../TFaaS
# generate proto files
protoc -I=$PWD/src/proto --cpp_out=$PWD/src/cpp $PWD/src/proto/tfaas.proto
cd src/cpp
mv tfaaspb.* TFClient

# adjust main.cc to use proper url and model
# adjust Makefile to use Makefile.mk.lxplus

# build codebase
make

# run C++ client
./tfaasClient
```
