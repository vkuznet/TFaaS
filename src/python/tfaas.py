#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : tfaas.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: TFaaS APIs for remote access of HEP data via uproot
"""

# system modules
import os
import sys
import json
import argparse
import traceback

# numpy modules
import numpy as np

# uproot modules
import uproot

# uproot reader
from reader import DataReader, make_plot, object_size, size_format

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input ROOT file")
        self.parser.add_argument("--params", action="store",
            dest="params", default="model.json", help="Input model parameters (default model.json)")
        self.parser.add_argument("--specs", action="store",
            dest="specs", default=None, help="Input specs file")
        self.parser.add_argument("--files", action="store",
            dest="files", default=None, help="either input file with files names or comma separate list of files")

class DataGenerator(object):
    def __init__(self, fin, params=None, specs=None):
        "Initialization function for Data Generator"
        if not params:
            params = {}
        # parse given parameters
        nan = params.get('nan', np.nan)
        nevts = params.get('nevts', -1)
        batch_size = params.get('batch_size', 256)
        verbose = params.get('verbose', 0)
        branch = params.get('branch', 'Events')
        branches = params.get('selected_branches', [])
        offset = params.get('offset', 1e-3)
        chunk_size = params.get('chunk_size', 1000)
        exclude_branches = params.get('exclude_branches', [])
        specs = params.get('specs', specs)

        if exclude_branches and not isinstance(exclude_branches, list):
            if os.path.isfile(exclude_branches):
                exclude_branches = \
                        [r.replace('\n', '') for r in open(exclude_branches).readlines()]
            else:
                exclude_branches = exclude_branches.split(',')
            print("exclude branches", exclude_branches)

        self.reader = DataReader(fin, branch=branch, selected_branches=branches,
            exclude_branches=exclude_branches, nan=nan, offset=offset,
            chunk_size=chunk_size, nevts=nevts, specs=specs, verbose=verbose)
        self.start_idx = 0
        self.nevts = nevts if nevts != -1 else self.reader.nrows
        self.stop_idx = params.get('nevts_chunk_size', 1000)
        self.batch_size = batch_size
        self.verbose = verbose

    def __len__(self):
        "Return total number of batches this generator can deliver"
        return int(np.floor(self.nevts / self.batch_size))

    def next(self):
        "Return next batch of events"
        if self.verbose:
            print("reading from {} to {}".format(self.start_idx, self.stop_idx))
        gen = self.read_data(self.start_idx, self.stop_idx)
        self.start_idx += self.stop_idx
        if self.start_idx >= self.reader.nrows: # reset our indicies
            self.start_idx = 0
            self.stop_idx = self.nevts
        data = []
        mask = []
        for (xdf, mdf) in gen:
            data.append(xdf)
            mask.append(mdf)
        return np.array(data), np.array(mask)

    def __iter__(self):
        "Provide iterator capabilities to the class"
        return self

    def __next__(self):
        "Provide generator capabilities to the class"
        return self.next()

    def read_data(self, start=0, stop=100, verbose=0):
        "Helper function to read ROOT data via uproot reader"
        if stop == -1:
            for _ in range(self.reader.nrows):
                xdf, mask = self.reader.next(verbose=verbose)
                yield (xdf, mask)
        else:
            for _ in range(start, stop):
                xdf, mask = self.reader.next(verbose=verbose)
                yield (xdf, mask)

class Trainer(object):
    def __init__(self, model, verbose=0):
        self.model = model
        self.verbose = verbose
        if self.verbose:
            print(self.model.summary())

    def fit(self, x_train, y_train, epochs, batch_size, shuffle=True, split=0.3):
        "Fit implementation of the trainer"
        if self.verbose:
            print("Perform fit on {} data with {} epochs, {} batch_size, {} split"\
                    .format(np.shape(x_train), epochs, batch_size, split))
        self.model.fit(x_train, y_train, epochs=epochs, batch_size=batch_size, shuffle=shuffle, validation_split=split, verbose=self.verbose)

    def predict(self):
        "Predict function of the trainer"
        pass

def xfile(fin):
    "Test if file is local or remote and setup proper prefix"
    if os.path.exists(fin):
        return fin
    return "root://cms-xrd-global.cern.ch/%s" % fin

def testModel(input_shape):
    "Simple ANN model for testing purposes"
    from keras.models import Sequential
    from keras.layers import Dense, Activation

    model = Sequential([
        Dense(32, input_shape=input_shape),
        Activation('relu'),
        Dense(2),
        Activation('softmax'),
    ])
    model.compile(optimizer='adam',
                  loss='categorical_crossentropy',
                  metrics=['accuracy'])
    return model

def test(files, params=None, specs=None):
    """
    Test function demonstrates workflow of setting up data generator and train the model
    over given set of files
    """
    from keras.utils import to_categorical
    if not params:
        params = {}
    if not specs:
        specs = {}
    for fin in files:
        fin = xfile(fin)
        print("Reading %s" % fin)
        gen = DataGenerator(fin, params, specs)
        epochs = specs.get('epochs', 10)
        batch_size = specs.get('batch_size', 50)
        shuffle = specs.get('shuffle', True)
        split = specs.get('split', 0.3)
        trainer = False
        for (x_train, _mask) in gen:
            if not trainer:
                input_shape = (np.shape(x_train)[-1],) # read number of attributes we have
                print("### input data: {}".format(input_shape))
                trainer = Trainer(testModel(input_shape), verbose=params.get('verbose', 0))
            print("x_train {} chunk of {} shape".format(x_train, np.shape(x_train)))
            if np.shape(x_train)[0] == 0:
                print("received empty x_train chunk")
                break
            # create dummy vector for y's for our x_train
            y_train = np.random.randint(2, size=np.shape(x_train)[0])
            y_train = to_categorical(y_train) # convert labesl to categorical values
            print("y_train {} chunk of {} shape".format(y_train, np.shape(y_train)))
            trainer.fit(x_train, y_train, epochs=epochs, batch_size=batch_size, shuffle=shuffle, split=split)

def testPyTorch(files, params=None, specs=None):
    """
    Test function demonstrates workflow of setting up data generator and train
    PyTorch model over given set of files
    """
    from jarray.pytorch import JaggedArrayLinear
    import torch
    if not params:
        params = {}
    if not specs:
        specs = {}
    for fin in files:
        fin = xfile(fin)
        print("Reading %s" % fin)
        gen = DataGenerator(fin, params, specs)
        epochs = specs.get('epochs', 10)
        batch_size = specs.get('batch_size', 50)
        shuffle = specs.get('shuffle', True)
        split = specs.get('split', 0.3)
        model = False
        for (x_train, x_mask) in gen:
            if not model:
                input_shape = np.shape(x_train)[-1] # read number of attributes we have
                print("### input data: {}".format(input_shape))
                model = torch.nn.Sequential(
                    JaggedArrayLinear(input_shape, 5),
                    torch.nn.ReLU(),
                    torch.nn.Linear(5, 1),
                )
                print(model)
            print("x_train chunk of {} shape".format(np.shape(x_train)))
            print("x_mask chunk of {} shape".format(np.shape(x_mask)))
            if np.shape(x_train)[0] == 0:
                print("received empty x_train chunk")
                break
            data = np.array([x_train, x_mask])
            preds = model(data).data.numpy()
            print("preds {} chunk of {} shape".format(preds, np.shape(preds)))

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    fin = opts.fin
    params = json.load(open(opts.params))
    specs = json.load(open(opts.specs)) if opts.specs else None
#     gen = DataGenerator(fin, params, specs)
#     print("Input source: %s, read %s events, can deliver %s batches" % (fin, gen.nevts, len(gen)))
    if os.path.isfile(opts.files):
        files = [f.replace('\n', '') for f in open(opts.files).readlines() if not f.startswith('#')]
    else:
        files = opts.files.split(',')
    testPyTorch(files, params, specs)
#     test(files, params, specs)

if __name__ == '__main__':
    main()
