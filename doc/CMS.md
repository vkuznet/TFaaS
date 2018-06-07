### TensorFlow as a Service (TFaaS) for CMS experiment

CMS experiment at CERN use various Machine Learning (ML) techniques, including
DNN, in various physics and computing related projects. We propose to
use TFaaS generic approach to serve ML models for CMS experiments.
The overall architecture is shown below:

![TFaaS Architecture](https://github.com/vkuznet/TFaaS/blob/master/images/TFaaS_architecture.png).

The projects R&D will explore the following topics
- build data-service which will read ROOT file(s) to train ML model
  - read ROOT files from python via [uproot](https://github.com/scikit-hep/uproot)
- serve ML model via REST API, aka *Machine Learning as a Service*
- read ML model either via 
  [CMSSW-DNN](https://gitlab.cern.ch/mrieger/CMSSW-DNN) framework or
  from external data-service (MLaaS or TFaaS) and demonstrate its usage for HEP
- explore Cloud and custom solution for TF back-end as well as
  [distributed Keras](https://github.com/cerndb/dist-keras) framework on Spark
  cluster
- port [fast.ai](http://www.fast.ai/)/[PyTorch](http://pytorch.org/) models
into [Keras](http://keras.io)/[TensorFlow](http://tensorflow.org). The PyTorch
framework provides dynamic models while TF are static one (compiled).
We need to find a way to port saved PyTorch models (in Protobuffer format)
into TF one in order to serve them in TFaaS. This has been discussed in
[fast.ai course](https://www.youtube.com/watch?v=9C06ZPF8Uuc&feature=youtu.be&t=30m10s)
as well as there is preliminary ideas
[here](https://briansp2020.github.io/2017/11/05/fast_ai_ROCm/)
and [here](https://github.com/briansp2020/courses).
See also discussion about [PyTorch vs TensorFlow](https://www.kdnuggets.com/2017/08/pytorch-tensorflow.html).

The TFaaS demonstrator instruction is available
[here](https://github.com/vkuznet/TFaaS/blob/master/DEMO.md)

