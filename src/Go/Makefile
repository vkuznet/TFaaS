TMPDIR:=/tmp/tfaas
GOPATH:=$(TMPDIR):${GOPATH}
OLDPWD:=$(PWD)
export GOPATH
OS := $(shell uname)
ifeq ($(OS),Darwin)
flags=-ldflags="-s -w"
else
# On Linux we still need TF shared libraries, therefore it's better to ask for
# static executable
# flags=-ldflags="-s -w -extldflags -static"
flags=-ldflags="-s -w"
endif
TAG := $(shell git tag | sort -r | head -n 1)

all: prepare build cleanup

prepare: cleanup
	mkdir -p $(TMPDIR)/src
	cp tfaas.go $(TMPDIR)
	cp -r tfaaspb $(TMPDIR)/src

cleanup:
	rm -rf $(TMPDIR)

build:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg; go build -o tfaas ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_all: prepare build_osx build_linux build_power8 build_arm64 cleanup

build_osx:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg tfaas_osx; GOOS=darwin go build -o tfaas_osx ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_linux:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg tfaas_linux; GOOS=linux go build -o tfaas_linux ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_power8:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg tfaas_power8; GOOS=ppc64le go build -o tfaas_power8 ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

build_arm64:
	sed -i -e "s,{{VERSION}},$(TAG),g" main.go
	go clean; rm -rf pkg tfaas_arm64; GOOS=arm64 go build -o tfaas_arm64 ${flags}
	sed -i -e "s,$(TAG),{{VERSION}},g" main.go

install:
	go install

clean:
	go clean; rm -rf pkg

test : test1

test1:
	cd test; go test
