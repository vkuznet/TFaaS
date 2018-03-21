### TensorFlow as a Service (TFaaS) for CMS experiment

CMS experiment at CERN use various Machine Learning (ML) techniques, including
DNN, in various physics and computing related projects. The popularity of
TensorFlow Google framework make it excellent choice to apply ML algorithms for
using in CMS workflow pipeline. The project intends to build end-to-end
data-service to serve TF trained model for CMSSW framework. The overall
architecture is shown below:

![TFaaS Architecture](https://github.com/vkuznet/TFaaS/blob/master/images/TFaaS_architecture.png).

The projects will explore and implement the following topics
- build data-service which will read ROOT file(s) to train ML model
  - read ROOT files from python via [uproot](https://github.com/scikit-hep/uproot)
- serve ML model via REST API
- write CMSSW analyzer which will read ML model via 
  [CMSSW-DNN](https://gitlab.cern.ch/mrieger/CMSSW-DNN) framework
  from external data-service and use it to make some predictions for physics topic
- explore Cloud and custom solution for TF back-end as well as
  [distributed Keras](https://github.com/cerndb/dist-keras) framework on Spark
  cluster

The TFaaS demonstrator instruction is available
[here](https://github.com/vkuznet/TFaaS/blob/master/DEMO.md)
