# Local setup, change as desired
SRC_DIR:=$(PWD)

# Compiler stuff
CPPFLAGS=-I$(SRC_DIR)
CXXFLAGS=-fPIC -pthread
LDFLAGS= -pthread
CXX:=g++
LINKER:=g++

# CPP unit stuff
# CPPUNIT_INCLUDES := -I/opt/local/include
# CPPUNIT_LIB_PATH := -L/opt/local/lib
CPPUNIT_INCLUDES := -I/afs/cern.ch/user/v/valya/workspace/protobuf-3.5.1/install/include
CPPUNIT_LIB_PATH := -L/afs/cern.ch/user/v/valya/workspace/protobuf-3.5.1/install/lib
CPPUNIT_LIB := $(CPPUNIT_LIB_PATH) -lprotoc -lprotobuf -lcurl
