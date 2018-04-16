# Local setup, change as desired
SRC_DIR:=$(PWD)

# Compiler stuff
CPPFLAGS=-I$(SRC_DIR)
CXXFLAGS=-fPIC -pthread
LDFLAGS= -pthread
CXX:=g++
LINKER:=g++

# CPP includes and libraries for TFaaS
CPPUNIT_INCLUDES := -I/opt/local/include
CPPUNIT_LIB_PATH := -L/opt/local/lib
CPPUNIT_LIB := $(CPPUNIT_LIB_PATH) -lprotoc -lprotobuf -lcurl
