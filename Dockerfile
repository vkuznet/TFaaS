FROM cmssw/cmsweb
MAINTAINER Valentin Kuznetsov vkuznet@gmail.com

# add environment
ENV WDIR=/data
ENV USER=tfaas

RUN yum update -y && yum clean all
RUN yum install -y git-core krb5-devel readline-devel openssl autoconf automake libtool make gcc gcc-c++ unzip
RUN yum clean all

# Create new user account
RUN useradd ${USER} && install -o ${USER} -d ${WDIR}
# add user to sudoers file
RUN echo "%$USER ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
# switch to user
USER ${USER}

# start the setup
RUN mkdir -p $WDIR
WORKDIR ${WDIR}

# download golang and install it
ENV GOPATH=$WDIR/gopath
RUN mkdir $WDIR/gopath
ENV PATH="${GOROOT}/bin:${WDIR}:${PATH}"

# install Go dependencies
RUN go get github.com/dmwm/cmsauth
RUN go get github.com/vkuznet/x509proxy
RUN go get github.com/sirupsen/logrus
RUN go get github.com/shirou/gopsutil

# download and insta TensorFlow libraries
# https://www.tensorflow.org/versions/master/install/install_go
ENV TF_LIB="libtensorflow-cpu-linux-x86_64-1.12.0.tar.gz"
RUN curl -k -L -O "https://storage.googleapis.com/tensorflow/libtensorflow/${TF_LIB}"
RUN tar xfz $TF_LIB
ENV LIBRARY_PATH="${LIBRARY_PATH}:${WDIR}/lib"
ENV LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:${WDIR}/lib"
RUN go get github.com/tensorflow/tensorflow/tensorflow/go
RUN go get github.com/tensorflow/tensorflow/tensorflow/go/op

# install protobuf
WORKDIR ${WDIR}
RUN git clone https://github.com/google/protobuf.git
WORKDIR ${WDIR}/protobuf
RUN ./autogen.sh
RUN ./configure --prefix=${WDIR}
RUN make
RUN make install
RUN go get -u github.com/golang/protobuf/protoc-gen-go

# build tfaas
WORKDIR ${WDIR}
RUN git clone https://github.com/vkuznet/TFaaS.git
WORKDIR $WDIR/TFaaS/src/Go
RUN make
ENV PATH="${WDIR}/TFaaS/src/Go:${PATH}"

# run the service
RUN mkdir models
CMD ["tfaas",  "-config", "config.json"]
