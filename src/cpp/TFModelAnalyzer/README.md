### Introduction
This area serve as an example how to write generic CMSSW EDAnalyzer class which
communicates with external data-service. The communication is done via
[protobuf](https://developers.google.com/protocol-buffers/) interface.  The
TFModelAnalyzer reads the geometry and extract hits from provided input ROOT
file. Then it submits POST request via [libcurl](https://curl.haxx.se/libcurl)
library to TFaaS server. The request contains a
[Hits](https://github.com/vkuznet/TFaaS/blob/master/src/proto/tfaas.proto)
data-structure and response has class predictions.

### Geometry
we can obtain the geometry file via the following command:
```
cmsRun
/cvmfs/cms.cern.ch/slc6_amd64_gcc530/cms/cmssw/CMSSW_8_0_19/src/Fireworks/Geometry/python/dumpRecoGeometry_cfg.py
tag=2015 out=geom2015.root
```
