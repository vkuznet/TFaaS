include Makefile.mk

UNAME:=$(shell uname -s)
ifeq ($(UNAME), Linux)
TFAAS_LIB := libTFClient.so
TFAAS_FLAGS := -shared
else
TFAAS_LIB := libTFClient.dylib
TFAAS_FLAGS := -dynamiclib
endif

CPPFLAGS=$(CPPUNIT_INCLUDES) -I$(SRC_DIR)/TFClient
all: prepare tflib tfaasClient

tfaasClient: tflib main.o
	$(LINKER) main.o $(LDFLAGS) $(CPPUNIT_LIB) -L$(SRC_DIR)/lib -lTFClient -o $@

fc_objects := $(patsubst %.cc,%.o,$(wildcard TFClient/*.cc))

tflib: $(fc_objects)
	$(LINKER) $+ $(LDFLAGS) $(TFAAS_FLAGS) $(CPPUNIT_LIB) -o $(SRC_DIR)/lib/$(TFAAS_LIB)

prepare:
	mkdir -p $(SRC_DIR)/lib

clean:
	rm -f *.o */*.o *.so
	rm -f $(fc_objects)
	rm -rf $(SRC_DIR)/lib
	rm tfaasClient
