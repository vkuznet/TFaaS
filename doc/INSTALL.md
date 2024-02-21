### TFaaS installation instructions
To build and install TFaaS the following dependencies are required:
1. Download and install TensorFlow libraries
```
# see https://www.tensorflow.org/versions/master/install/install_go
#     https://www.tensorflow.org/install/lang_c
TF_LIB="libtensorflow-cpu-linux-x86_64-2.15.0.tar.gz"
curl -k -L -O "https://storage.googleapis.com/tensorflow/libtensorflow/${TF_LIB}"
tar xfz $TF_LIB
# if you have local TF area you should compile code with the following settings
CGO_CFLAGS='-I/path/tensorflow/include' CGO_LDFLAGS='-L/path/tensorflow/lib' make
```

Next two steps are only required for older GO releases without go.mod support:
2. (optional|obsolete): install Go language and required dependencies
```
go get github.com/dmwm/cmsauth
go get github.com/vkuznet/x509proxy
go get github.com/sirupsen/logrus
go get github.com/shirou/gopsutil
go get github.com/tensorflow/tensorflow/tensorflow/go
go get github.com/tensorflow/tensorflow/tensorflow/go/op
```

3. (optional|obsolete) Install ProtoBuffer
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
# use make if TF libs/includes are available on the system
make

# or use the following command, replacing the /path with your own PATH
# which should point to local TF installation
CGO_CFLAGS='-I/path/tensorflow/include' CGO_LDFLAGS='-L/path/tensorflow/lib' make
```

5. Enjoy
```
# please note if you install local version of TF then you should properly setup
# LD_LIBRARY_PATH on your system to point to local area of TF libs, e.g.
export LD_LIBRARY_PATH=/path/tensorflow/lib

# to run your TFaaS simply run it with your configuration:
./tfaas -config config.json
```
