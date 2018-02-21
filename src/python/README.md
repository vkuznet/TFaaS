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
This code allows to read ROOT file content directly into NumPy/Pandas dataframe.
It is based on [uproot](https://github.com/scikit-hep/uproot) framework.
Basic usage
```
./tfaas.py --help
usage: PROG [-h] [--fin FIN] [--branch BRANCH] [--branches BRANCHES]
            [--list-branches] [--fout FOUT] [--verbose]

optional arguments:
  -h, --help           show this help message and exit
  --fin FIN            Input ROOT file
  --branch BRANCH      Input ROOT file branch (default Events)
  --branches BRANCHES  ROOT branches to read, 'Electron_,Jet_'
  --list-branches      list ROOT branches and exit
  --fout FOUT          Output model file
  --verbose            verbose output
```

To inspect the ROOT file please use `--list-branches` option, e.g.
```
./tfaas.py --fin=/opt/cms/data/nano-RelValTTBar.root --list-branches
### Branch LuminosityBlocks
run
luminosityBlock
### Branch Runs
run
genEventCount
genEventSumw
genEventSumw2
nLHEScaleSumw
LHEScaleSumw
nLHEPdfSumw
LHEPdfSumw
### Branch Events
run
luminosityBlock
event
nElectron
Electron_deltaEtaSC
Electron_dxy
Electron_eta
Electron_mass
...
```

And here is an example of readying Electron branches into pandas DataFrame:
```
./tfaas.py --fin=/opt/cms/data/nano-RelValTTBar.root --branch=Events --branches="Electron_pt,Electron_eta,Electron_dxy"
      Electron_dxy  Electron_eta  Electron_pt
0         0.003125     -1.424316    11.108109
1        -0.448377     -1.196289   120.067390
2         1.136900     -1.802734    19.996458
3         0.014889      0.193848     8.102822
...
```

More examples about using uproot may be found
[here](https://github.com/jpivarski/jupyter-talks/blob/master/2017-10-13-lpc-testdrive/uproot-introduction-evaluated.ipynb)
and
[here](https://github.com/jpivarski/jupyter-talks/blob/master/2017-10-13-lpc-testdrive/nested-structures-evaluated.ipynb)
