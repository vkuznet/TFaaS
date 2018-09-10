### TFaaS client
We provide pure python
[client](https://github.com/vkuznet/TFaaS/blob/master/src/python/tfaas_client.py)
to perform all necessary action against TFaaS server. Here is short
description of available APIs:

```
# setup url to point to your TFaaS server
url=http://localhost:8083

# create upload json file, which should include
# fully qualified model file name
# fully qualified labels file name
# model name you want to assign to your model file
# fully qualified parameters json file name
# For example, here is a sample of upload json file
{
    "model": "/path/model_0228.pb",
    "labels": "/path/labels.txt",
    "name": "model_name",
    "params":"/path/params.json"
}

# upload given model to the server
tfaas_client.py --url=$url --upload=upload.json

# list existing models in TFaaS server
tfaas_client.py --url=$url --models

# delete given model in TFaaS server
tfaas_client.py --url=$url --delete=model_name

# prepare input json file for querying model predictions
# here is an example of such file
{"keys":["attribute1", "attribute2"], values: [1.0, -2.0]}

# get predictions from TFaaS server
tfaas_client.py --url=$url --predict=input.json

# get image predictions from TFaaS server
# here we refer to uploaded on TFaaS ImageModel model
tfaas_client.py --url=$url --image=/path/file.png --model=ImageModel
```

### HEP resnet
We provided full code called `hep_resnet.py` as a basic model based on
[ResNet](https://github.com/raghakot/keras-resnet) implementation.
It can classify images from HEP events, e.g.
```
hep_resnet.py --fdir=/path/hep_images --flabels=labels.csv --epochs=200 --mdir=models
```
Here we supply input directory `/path/hep_images` which contains HEP images
in `train` folder along with `labels.csv` file which provides labels.
The model runs for 200 epochs and save Keras/TF model into `models` output
directory.

### Reading ROOT files
TFaaS python repository provides two base modules to read and manipulate with
HEP ROOT files. The `reader.py` module defines a DataReader class which is
able to read either local or remote ROOT files (via xrootd). And, `tfaas.py`
module provide a basic DataGenerator class which can be used with any ML
framework to read HEP ROOT data in chunks. Both modules are based on
[uproot](https://github.com/scikit-hep/uproot) framework.

Basic usage
```
./reader.py --help
usage: PROG [-h] [--fin FIN] [--fout FOUT] [--nan NAN] [--branch BRANCH]
            [--identifier IDENTIFIER] [--branches BRANCHES]
            [--exclude-branches EXCLUDE_BRANCHES] [--nevts NEVTS]
            [--chunk-size CHUNK_SIZE] [--specs SPECS] [--offset OFFSET]
            [--info] [--hists] [--verbose VERBOSE]

optional arguments:
  -h, --help            show this help message and exit
  --fin FIN             Input ROOT file
  --fout FOUT           Output file name to write ROOT specs
  --nan NAN             NaN value (default 0)
  --branch BRANCH       Input ROOT file branch (default Events)
  --identifier IDENTIFIER
                        Event identifier (default run,event,luminosityBlock)
  --branches BRANCHES   Comma separated list of branches to read (default all)
  --exclude-branches EXCLUDE_BRANCHES
                        Comma separated list of branches to exclude (default
                        None)
  --nevts NEVTS         number of events to parse (default 5, use -1 to read
                        all events)
  --chunk-size CHUNK_SIZE
                        Chunk size to use (default 1000)
  --specs SPECS         Input specs file
  --offset OFFSET       Offset value to shift from nan (default 1e-3)
  --info                Provide info about ROOT tree
  --hists               Create historgams for ROOT tree
  --verbose VERBOSE     verbosity level

# here is a concrete example of reading local ROOT file:
./reader.py --fin=/opt/cms/data/Tau_Run2017F-31Mar2018-v1_NANOAOD.root --info --verbose=1 --nevts=2000

# here is an example of reading remote ROOT file:
./reader.py --fin=root://cms-xrd-global.cern.ch//store/data/Run2017F/Tau/NANOAOD/31Mar2018-v1/20000/6C6F7EAE-7880-E811-82C1-008CFA165F28.root --verbose=1 --nevts=2000 --info

# both of aforementioned commands produce the following output
First pass: 2000 events, 35.4363200665 sec, shape (2316,) 648 branches: flat 232 jagged
VMEM used: 960.479232 (MB) SWAP used: 0.0 (MB)
Number of events  : 1131872
# flat branches   : 648
...  # followed by a long list of ROOT branches found along with their dimentionality
TrigObj_pt values in [5.03515625, 1999.75] range, dim=21
```

The `tfaas.py` module is intended to be used in TFaaS server which can
read remote files and perform the training of ML models with HEP ROOT
files.

More examples about using uproot may be found
[here](https://github.com/jpivarski/jupyter-talks/blob/master/2017-10-13-lpc-testdrive/uproot-introduction-evaluated.ipynb)
and
[here](https://github.com/jpivarski/jupyter-talks/blob/master/2017-10-13-lpc-testdrive/nested-structures-evaluated.ipynb)
