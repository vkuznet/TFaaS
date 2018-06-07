### TFaaS installation instructions
To build and install TFaaS the following dependencies are required:
1. Download and install TensorFlow libraries
```
# see https://www.tensorflow.org/versions/master/install/install_go
TF_LIB="libtensorflow-cpu-linux-x86_64-1.6.0.tar.gz"
curl -k -L -O "https://storage.googleapis.com/tensorflow/libtensorflow/${TF_LIB}"
tar xfz $TF_LIB
# then setup your LD_LIBRARY_PATH to include your libtensorflow path
```

2. install Go language and required dependencies
```
go get github.com/dmwm/cmsauth
go get github.com/vkuznet/x509proxy
go get github.com/sirupsen/logrus
go get github.com/shirou/gopsutil
go get github.com/tensorflow/tensorflow/tensorflow/go
go get github.com/tensorflow/tensorflow/tensorflow/go/op
```

3. Install ProtoBuffer
```
git clone https://github.com/google/protobuf.git
cd protobuf
./autogen.sh
./configure --prefix=${WDIR}
make
make install
go get -u github.com/golang/protobuf/protoc-gen-go
```

4. Build tfaas code
```
git clone https://github.com/vkuznet/TFaaS.git
cd TFaaS/src/Go
make
```

5. Enjoy
```
# to run your TFaaS simply run it with your configuration:
./tfaas -config config.json
```
