include Makefile.mk

CPPFLAGS=$(CPPUNIT_INCLUDES) -I$(SRC_DIR)/TFClient
all: prepare libTFClient.so tfaasClient

tfaasClient: libTFClient.so main.o
	$(LINKER) main.o $(LDFLAGS) $(CPPUNIT_LIB) -L$(SRC_DIR)/lib -lTFClient -o $@

fc_objects := $(patsubst %.cc,%.o,$(wildcard TFClient/*.cc))

libTFClient.so: $(fc_objects)
	$(LINKER) $+ -shared -o $(SRC_DIR)/lib/$@

prepare:
	mkdir -p $(SRC_DIR)/lib

clean:
	rm -f *.o */*.o *.so
	rm -f $(fc_objects)
	rm -rf $(SRC_DIR)/lib
	rm tfaasClient
