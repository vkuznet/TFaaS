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

class DataGenerator(object):
    def __init__(self, fin, params, specs=None):
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
        self.stop_idx = params.get('nevts', self.nevts)
        self.batch_size = batch_size

    def __len__(self):
        "Return total number of batches this generator can deliver"
        return int(np.floor(self.nevts / self.batch_size))

    def __getitem__(self):
        "Return next batch of events"
        data = self.read_data(self.start_idx, self.stop_idx)
        self.start_idx += self.stop_idx
        if self.start_idx >= self.reader.nrows: # reset our indicies
            self.start_idx = 0
            self.stop_idx = self.nevts
        yield data

    def read_data(self, start=0, stop=100, verbose=0):
	"Helper function to read ROOT data via uproot reader"
	if stop == -1:
	    for _ in range(self.reader.nrows):
		xdf = self.reader.next(verbose=verbose)
		yield xdf
	else:
	    for _ in range(start, stop):
		xdf = self.reader.next(verbose=verbose)
		yield xdf

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    fin = opts.fin
    params = json.load(open(opts.params))
    specs = json.load(open(opts.specs)) if opts.specs else None
    gen = DataGenerator(fin, params, specs)
    print("Input source: %s, read %s events, can deliver %s batches" % (fin, gen.nevts, len(gen)))

if __name__ == '__main__':
    main()
