FROM golang:latest as go-builder
MAINTAINER Valentin Kuznetsov vkuznet@gmail.com

# TF libraries can be found at https://www.tensorflow.org/install/lang_c
# Go TF build is here: https://github.com/tensorflow/build/tree/master/golang_install_guide
# ENV TFVER=2.4.0
ENV TFVER=2.11.0
RUN curl -ksLO "https://storage.googleapis.com/tensorflow/libtensorflow/libtensorflow-cpu-linux-x86_64-${TFVER}.tar.gz" && \
    tar xfz libtensorflow-cpu-linux-x86_64-${TFVER}.tar.gz && \
    cp -a lib/* /usr/local/lib && cp -a include/* /usr/local/include
ENV LD_LIBRARY_PATH="${LD_LIBRARY_PATH}:/go/lib"

# download golang and install it
ENV GOPATH=/go/gopath
ENV PATH="${GOROOT}/bin:/go/gopath/bin:${PATH}"

# build tfaas
# we'll use tfgo build instead of offical TF one, see
# https://github.com/tensorflow/tensorflow/issues/41808
# https://github.com/tensorflow/tensorflow/issues/48017
# https://github.com/tensorflow/tensorflow/issues/35133#issuecomment-807404740
# https://github.com/galeone/tfgo
#RUN go env -w GONOSUMDB="github.com/galeone/tensorflow" && \
#    go get github.com/galeone/tfgo && \
#    go get github.com/dmwm/cmsauth && \
#    go get github.com/vkuznet/x509proxy && \
#    go get github.com/sirupsen/logrus && \
#    go get github.com/shirou/gopsutil

RUN git clone https://github.com/vkuznet/TFaaS.git && \
    cd TFaaS/src/Go && \
    rm go.mod go.sum && go mod init github.com/vkuznet/TFaaS && go mod tidy && \
    make

# final image
FROM debian:stretch
RUN mkdir -p /data/lib
COPY --from=go-builder /go/TFaaS/src/Go /data/
COPY --from=go-builder /go/lib /data/lib
RUN mkdir /data/models
ENV WDIR=/data
ENV LIBRARY_PATH="${WDIR}/lib"
ENV LD_LIBRARY_PATH="${WDIR}/lib"
ENV PATH="${WDIR}:${PATH}"

# run the service
WORKDIR ${WDIR}
CMD ["tfaas",  "-config", "config.json"]
