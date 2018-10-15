#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : reader.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: Reader for uproot
Access file via xrootd
    xrdcp root://cms-xrd-global.cern.ch/<lfn>
for example:
    xrdcp root://cms-xrd-global.cern.ch//store/data/Run2017F/Tau/NANOAOD/31Mar2018-v1/20000/6C6F7EAE-7880-E811-82C1-008CFA165F28.root
"""
from __future__ import print_function, division, absolute_import

# system modules
import os
import sys
import time
import json
import random
import argparse
import traceback

# numpy
import numpy as np

# uproot
import uproot
try:
    # uproot verion 3.X
    from awkward import JaggedArray
except:
    # uproot verion 2.X
    from uproot.interp.jagged import JaggedArray

# numba
try:
    from numba import jit
except:
    def jit(f):
        "Simple decorator which calls underlying function"
        def new_f():
            "Action function"
            f()
        return new_f

# psutil
try:
    import psutil
except:
    psutil = None

# histogrammar
try:
    import histogrammar as hg
    import matplotlib
    matplotlib.use('Agg')
    from matplotlib.backends.backend_pdf import PdfPages
    import matplotlib.pyplot as plt
except ImportError:
    hg = None

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input ROOT file")
        self.parser.add_argument("--fout", action="store",
            dest="fout", default="", help="Output file name for ROOT specs")
        self.parser.add_argument("--nan", action="store",
            dest="nan", default=np.nan, help="NaN value for padding, default np.nan")
        self.parser.add_argument("--branch", action="store",
            dest="branch", default="Events", help="Input ROOT file branch, default Events")
        self.parser.add_argument("--identifier", action="store",
            dest="identifier", default="run,event,luminosityBlock", help="Event identifier, default run,event,luminosityBlock")
        self.parser.add_argument("--branches", action="store",
            dest="branches", default="", help="Comma separated list of branches to read, default all")
        self.parser.add_argument("--exclude-branches", action="store",
            dest="exclude_branches", default="", help="Comma separated list of branches to exclude, default None")
        self.parser.add_argument("--nevts", action="store",
            dest="nevts", default=5, help="number of events to parse, default 5, use -1 to read all events)")
        self.parser.add_argument("--chunk-size", action="store",
            dest="chunk_size", default=1000, help="Chunk size to use, default 1000")
        self.parser.add_argument("--specs", action="store",
            dest="specs", default=None, help="Input specs file")
        self.parser.add_argument("--redirector", action="store",
            dest="redirector", default='root://cms-xrd-global.cern.ch',
            help="XrootD redirector, default root://cms-xrd-global.cern.ch")
        self.parser.add_argument("--info", action="store_true",
            dest="info", default=False, help="Provide info about ROOT tree")
        self.parser.add_argument("--hists", action="store_true",
            dest="hists", default=False, help="Create historgams for ROOT tree")
        self.parser.add_argument("--verbose", action="store",
            dest="verbose", default=0, help="verbosity level")

def dump_histograms(hdict, hgkeys):
    "Helper function to dump histograms"
    if not hg:
        return
    for key in hgkeys:
        make_plot(hdict['%s_orig' % key], '%s_orig' % key)
        make_plot(hdict['%s_norm' % key], '%s_norm' % key)

def make_plot(hist, name):
    "Helper function to make histogram"
    pdir = os.path.join(os.getcwd(), 'pdfs')
    try:
        os.makedirs(pdir)
    except OSError:
        pass
    fname = os.path.join(pdir, '%s.pdf' % name)
    pdf = PdfPages(fname)
    fig = plt.figure()
    hist.plot.matplotlib(name=name)
    pdf.savefig(fig)
    plt.close(fig)
    pdf.close()

def mem_usage(vmem0, swap0, vmem1, swap1, msg=None):
    "helper function to show memory usage"
    if msg:
        print(msg)
    mbyte = 10**6
    vmem_used = (vmem1.used-vmem0.used)/mbyte
    swap_used = (swap1.used-swap0.used)/mbyte
    print("VMEM used: %s (MB) SWAP used: %s (MB)" % (vmem_used, swap_used))

def performance(nevts, tree, data, startTime, endTime, msg=""):
    "helper function to show performance metrics of data read from a given tree"
    try:
        nbytes = sum(x.content.nbytes + x.stops.nbytes \
                if isinstance(x, JaggedArray) \
                else x.nbytes for x in data.values())
        print("# %s entries, %s %sbranches, %s MB, %s sec, %s MB/sec, %s kHz" % \
                (
            nevts,
            len(data),
            msg,
            nbytes / 1024**2,
            endTime - startTime,
            nbytes / 1024**2 / (endTime - startTime),
            tree.numentries / (endTime - startTime) / 1000))
    except Exception as exc:
        print(str(exc))

def steps(total, size):
    "Return list of steps within total number given events"
    step = int(float(total)/float(size))
    chunk = []
    for idx in range(total):
        if len(chunk) == size:
            yield chunk
            chunk = []
        chunk.append(idx)
    if len(chunk) > 0:
        yield chunk

def dim_jarr(arr):
    "Return dimention (max length) of jagged array"
    jdim = 0
    for item in arr:
        if jdim < len(item):
            jdim = len(item)
    return jdim

def min_max_arr(arr):
    """
    Helper function to find out min/max values of given array.
    The array can be either jagged one or normal numpy.ndarray
    """
    try:
        if isinstance(arr, JaggedArray):
            minv = 1e15
            maxv = -1e15
            for item in arr:
                if len(item) == 0:
                    continue
                if np.min(item) < minv:
                    minv = np.min(item)
                if np.max(item) > maxv:
                    maxv = np.max(item)
            return float(minv), float(maxv)
        return float(np.min(arr)), float(np.max(arr))
    except ValueError:
        return 1e15, -1e15

class DataReader(object):
    """
    DataReader class provide interface to read ROOT files
    and APIs to access its data. It uses two-pass procedure
    unless specs file is provided. The first pass parse entire
    file and identifies flat/jagged keys, their dimensionality
    and min/max values. All of them are stored in a file specs.
    The second pass uses specs to convert jagged structure of
    ROOT file into flat DataFrame format.
    """
    def __init__(self, fin, branch='Events', selected_branches=None,
            exclude_branches=None, identifier=['run', 'event', 'luminosityBlock'],
            chunk_size=1000, nevts=-1, specs=None, nan=np.nan, histograms=False,
            redirector='root://cms-xrd-global.cern.ch', verbose=0):
        self.fin = xfile(fin, redirector)
        self.verbose = verbose
        if verbose:
            print("Reading {}".format(self.fin))
        self.istream = uproot.open(self.fin)
        self.branches = {}
        self.gen = None
        self.out_branches = []
        self.identifier = identifier
        self.tree = self.istream[branch]
        self.nrows = self.tree.numentries
        self.nevts = nevts
        self.idx = -1
        self.chunk_idx = 0
        self.lastIdx = -1
        self.chunk_size = chunk_size if chunk_size < self.nrows else self.nrows
        self.nan = float(nan)
        self.attrs = []
        self.shape = None
        self.cache = {}
        self.hdict = {}
        self.hists = histograms
        if specs:
            self.load_specs(specs)
        else:
            self.jdim = {}
            self.minv = {}
            self.maxv = {}
            self.jkeys = []
            self.fkeys = []
            self.nans = {}

        # perform initialization
        time0 = time.time()
        self.init()
        if self.verbose:
            print("{} init is complete in {} sec".format(self, time.time()-time0))

        if selected_branches:
#             self.out_branches = [a for b in selected_branches for a in self.attrs if a.startswith(b)]
            self.out_branches = []
            for attr in self.attrs:
                for name in selected_branches:
                    if name.find('*') != -1:
                        if attr.startswith(name):
                            self.out_branches.append(attr)
                    else:
                        if attr == name:
                            self.out_branches.append(attr)

            if self.out_branches:
                if self.verbose:
                    print("Select branches ...")
                    for name in sorted(self.out_branches):
                        print(name)
        if exclude_branches:
            out_branches = set()
            for attr in self.attrs:
                count = 0
                for name in exclude_branches:
                    if name.find('*') != -1:
                        if attr.startswith(name):
                            count += 1
                    else:
                        if attr == name:
                            count += 1
                if not count:
                    out_branches.add(attr)
            self.out_branches = list(out_branches)
            if self.out_branches:
                if self.verbose:
                    print("Select branches ...")
                    for name in sorted(self.out_branches):
                        print(name)

        # declare histograms for original and normilized values
        if hg and self.hists:
            for key in self.attrs:
                low = self.minv[key]
                high = self.maxv[key]
                self.hdict['%s_orig' % key] = \
                        hg.Bin(num=100, low=low, high=high, quantity=lambda x: x, value=hg.Count())
                self.hdict['%s_norm' % key] = \
                        hg.Bin(num=100, low=0, high=1, quantity=lambda x: x, value=hg.Count())


    def load_specs(self, specs):
        "load given specs"
        if not isinstance(specs, dict):
            if self.verbose:
                print("load specs from {}".format(specs))
            specs = json.load(open(specs))
        if self.verbose > 1:
            print("ROOT specs: {}".format(json.dumps(specs)))
        self.jdim = specs['jdim']
        self.minv = specs['minv']
        self.maxv = specs['maxv']
        self.jkeys = specs['jkeys']
        self.fkeys = specs['fkeys']
        self.nans = specs['nans']

    def fetch_data(self, key):
        "fetch data for given key from underlying ROOT tree"
        if key in self.branches:
            return self.branches[key]
        raise Exception('Unable to find %s key in ROOT branches' % key)

    def read_chunk(self, nevts, set_branches=False, set_min_max=False):
        "Reach chunk of events and determine min/max values as well as load branch values"
        # read some portion of the data to determine branches
        startTime = time.time()
        if not self.gen:
            if self.out_branches:
                self.gen = self.tree.iterate(branches=self.out_branches+self.identifier,
                        entrysteps=nevts, keycache=self.cache)
            else:
                self.gen = self.tree.iterate(entrysteps=nevts, keycache=self.cache)
        self.branches = {} # start with fresh dict
        try:
            self.branches = next(self.gen) # python 3.X and 2.X
        except StopIteration:
            if self.out_branches:
                self.gen = self.tree.iterate(branches=self.out_branches+self.identifier,
                        entrysteps=nevts, keycache=self.cache)
            else:
                self.gen = self.tree.iterate(entrysteps=nevts, keycache=self.cache)
            self.branches = next(self.gen) # python 3.X and 2.X
        endTime = time.time()
        self.idx += nevts
        if self.verbose:
            performance(nevts, self.tree, self.branches, startTime, endTime)
        if set_branches:
            for key, val in self.branches.items():
                self.minv[key], self.maxv[key] = min_max_arr(val)
                if isinstance(val, JaggedArray):
                    self.jkeys.append(key)
                else:
                    self.fkeys.append(key)
        if set_min_max:
            for key, val in self.branches.items():
                minv, maxv = min_max_arr(val)
                if minv < self.minv[key]:
                    self.minv[key] = minv
                if maxv > self.maxv[key]:
                    self.maxv[key] = maxv

    def columns(self):
        "Return columns of produced output vector"
        cols = self.flat_keys()
        for key in self.jagged_keys():
            for idx in range(self.jdim[key]):
                cols.append('%s_%s' % (key, idx))
        return cols

    def init(self):
        "Initialize class data members by scaning ROOT tree"
        if self.jdim and self.minv and self.maxv:
            self.attrs = sorted(self.flat_keys()) + sorted(self.jagged_keys())
            self.shape = len(self.flat_keys()) + sum(self.jdim.values())
            msg = "+++ first pass: %s events, (%s-flat, %s-jagged) branches, %s attrs" \
                    % (self.nrows, len(self.flat_keys()), len(self.jagged_keys()), self.shape)
            if self.verbose:
                print(msg)
            self.idx = -1
            return

        if psutil and self.verbose:
            vmem0 = psutil.virtual_memory()
            swap0 = psutil.swap_memory()

        msg = ''
        time0 = time.time()

        # scan all rows to find out largest jagged array dimension
        tot = 0
        set_branches = True
        set_min_max = True
        for chunk in steps(self.nrows, self.chunk_size):
            nevts = len(chunk) # chunk here contains event indexes
            tot += nevts
            self.read_chunk(nevts, set_branches=set_branches, set_min_max=set_min_max)
            set_branches = False # we do it once
            for key in self.jkeys:
                if key not in self.jdim:
                    self.jdim[key] = 0
                dim = dim_jarr(self.fetch_data(key))
                if dim > self.jdim.get(key, 0):
                    self.jdim[key] = dim
            if self.nevts > 0 and tot > self.nevts:
                break

        self.nevts = tot

        # initialize all nan values (zeros) in normalize phase-space
        # this should be done after we get all min/max values
        for key in self.branches.keys():
            self.nans[key] = self.normalize(key, 0)

        # reset internal indexes since we done with first pass reading
        self.idx = -1
        self.gen = None

        # define final list of attributes
        self.attrs = sorted(self.flat_keys()) + sorted(self.jagged_keys())

        if self.verbose > 1:
            print("\n### Dimensionality")
            for key, val in self.jdim.items():
                print(key, val)
            print("\n### min/max values")
            for key, val in self.minv.items():
                print(key, val, self.maxv[key])
        self.shape = len(self.flat_keys()) + sum(self.jdim.values())
        msg = "--- first pass: %s events, (%s-flat, %s-jagged) branches, %s attrs" \
                % (self.nrows, len(self.flat_keys()), len(self.jagged_keys()), self.shape)
        if self.verbose:
            print(msg)
        if psutil and self.verbose:
            vmem1 = psutil.virtual_memory()
            swap1 = psutil.swap_memory()
            mem_usage(vmem0, swap0, vmem1, swap1)

    def write_specs(self, fout):
        "Write specs about underlying file"
        out = {'jdim': self.jdim, 'minv': self.minv, 'maxv': self.maxv}
        out['fkeys'] = self.flat_keys()
        out['jkeys'] = self.jagged_keys()
        out['nans'] = self.nans
        if self.verbose:
            print("write specs {}".format(fout))
        with open(fout, 'w') as ostream:
            ostream.write(json.dumps(out))

    def next(self, verbose=0):
        "Provides read interface for next event using vectorize approach"
        self.idx = self.idx + 1
        # build output matrix
        time0 = time.time()
        shape = len(self.flat_keys())
        for key in sorted(self.jagged_keys()):
            shape += self.jdim[key]
        xdf = np.ndarray(shape=(shape,))
        mask = np.ndarray(shape=(shape,), dtype=np.int)

        # read new chunk of records if necessary
        if not self.idx % self.chunk_size:
            if self.idx + self.chunk_size > self.nrows:
                nevts = self.nrows - self.idx
            else:
                nevts = self.chunk_size
            self.read_chunk(nevts)
            self.chunk_idx = 0 # reset chunk index after we read the chunk of data
            self.idx = self.idx - nevts # reset index after chunk read by nevents offset
            if self.verbose > 1:
                print("idx", self.idx, "read", nevts, "events")

        # read event info
        event = []
        for key in self.identifier:
            fdata = self.fetch_data(key)
            if len(fdata) <= self.chunk_idx:
                raise Exception("For key='%s' unable to find data at pos=%s while got %s" \
                        % (key, self.chunk_idx, len(fdata)))
            event.append(fdata[self.chunk_idx])

        # form DataFrame record
        rec = {}
        for key in self.branches.keys():
            try:
                fdata = self.fetch_data(key)
                if len(fdata) <= self.chunk_idx:
                    raise Exception("For key='%s' unable to find data at pos=%s while got %s" \
                            % (key, self.chunk_idx, len(fdata)))
                rec[key] = fdata[self.chunk_idx]
            except:
                print("failed key", key)
                print("failed idx", self.chunk_idx)
                print("len(fdata)", len(fdata))
                raise

        # advance chunk index since we read the record
        self.chunk_idx = self.chunk_idx + 1

        idx = 0
        for idx, key in enumerate(sorted(self.flat_keys())):
            if sys.version.startswith('3.') and isinstance(key, str):
                key = key.encode('ascii') # convert string to binary
            xdf[idx] = self.normalize(key, rec[key])
            if hg and self.hists:
                self.hdict['%s_orig' % key].fill(rec[key])
                if xdf[idx] != self.nan:
                    self.hdict['%s_norm' % key].fill(xdf[idx])
            mask[idx] = 1
        if idx: # only advance position if we read something from flat_keys
            pos = idx + 1 # position in xdf for jagged branches
        else:
            pos = 0

        for key in sorted(self.jagged_keys()):
            # check if key in our record
            if key in rec.keys():
                vals = rec.get(key, [])
            else: # if not convert key to bytes key and use it to look-up a value
                vals = rec.get(key.encode('utf-8'), [])
            for jdx in range(self.jdim[key]):
                # assign np.nan in case if we get empty array
                val = vals[jdx] if len(vals) > jdx else np.nan
                idx = pos+jdx
                xdf[idx] = self.normalize(key, val)
                if hg and self.hists:
                    self.hdict['%s_orig' % key].fill(val)
                    if xdf[idx] != self.nan:
                        self.hdict['%s_norm' % key].fill(xdf[idx])
                if np.isnan(val):
                    mask[idx] = 0
                else:
                    mask[idx] = 1
            pos = idx + 1

        if verbose > 1:
            print("# idx=%s event=%s shape=%s proc.time=%s" % (
                self.idx, event, np.shape(xdf), (time.time()-time0)))
            if self.idx < 3:
                # pick-up 3 branches for cross checking
                if len(self.jagged_keys()):
                    arrIdx = [random.randint(0, len(self.jagged_keys())-1) for _ in range(3)]
                    try:
                        keys = [self.jagged_keys()[i] for i in arrIdx]
                        for key in keys:
                            data = self.tree[key].array()
                            idx = self.attrs.index(key)
                            startIdx, endIdx = self.find_branch_idx(key)
                            print("+ branch=%s, row %s, position %s:%s, min=%s max=%s" \
                                    % (key, self.idx, startIdx, endIdx, self.minv[key], self.maxv[key]))
                            print("+ xdf", xdf[startIdx:endIdx])
                            print(data)
                    except:
                        print("arrIdx=%s, len(jagged_keys)=%s" % (arrIdx, len(self.jagged_keys())))
                        traceback.print_exc()
        return xdf, mask

    def find_branch_idx(self, attr):
        "Find start and end indexes of given attribute"
        idx = self.attrs.index(attr)
        if attr in self.flat_keys():
            return idx, idx+1
        start_idx = len(self.flat_keys())
        for key in sorted(self.jagged_keys()):
            if key == attr:
                return start_idx, start_idx + self.jdim[key]
            start_idx += self.jdim[key]
        raise Exception("Unable to find branch idx for %s" % attr)

    def jagged_keys(self):
        "helper function to return list of jagged branches"
        jkeys = sorted(list(self.jkeys))
        if self.out_branches:
            return [k for k in jkeys if k in self.out_branches]
        return jkeys

    def flat_keys(self):
        "helper function to return list of normal branches"
        if self.out_branches:
            fkeys = [k for k in self.fkeys if k not in self.identifier and k in self.out_branches]
            return sorted(fkeys)
        fkeys = [k for k in self.fkeys if k not in self.identifier]
        return sorted(fkeys)

    def draw_value(self, key):
        "Draw a random value from underlying chunk for a given key"
        data = self.branches[key] # jagged array
        # get random index for accessing element of jagged array
        while True:
            idx = random.randint(0, len(data)-1)
            values = data[idx]
            if len(values):
                if len(values) == 1:
                    val = values[0]
                else:
                    jdx = random.randint(0, len(values)-1)
                    val = values[jdx]
                if random.randint(0, 1):
                    return val + val/10.
                return val - val/10.

    def normalize(self, key, val):
        "Normalize given value to 0-1 range according to min/max values"
        # in case if our value is np.nan we'll assign nan value given to class
        if np.isnan(val):
            return self.nan
        minv = float(self.minv.get(key, 0))
        maxv = float(self.maxv.get(key, 1))
        if maxv == minv:
            return val
        return (val-minv)/(maxv-minv)

    def denormalize(self, key, val):
        "De-normalize given value to 0-1 range according to min/max values"
        if val == 0:
            return self.nan
        minv = float(self.minv.get(key, 0))
        maxv = float(self.maxv.get(key, 1))
        return val*(maxv-minv)+minv

    def info(self):
        "Provide human readable form of ROOT branches"
        print("Number of events  : %s" % self.nrows)
        print("# flat branches   : %s" % len(self.flat_keys()))
        if self.verbose:
            for key in self.flat_keys():
                print("%s values in [%s, %s] range, dim=%s" % (
                    key,
                    self.minv.get(key, 'N/A'),
                    self.maxv.get(key, 'N/A'),
                    self.jdim.get(key, 'N/A')))
        print("# jagged branches : %s" % len(self.jagged_keys()))
        if self.verbose:
            for key in self.jagged_keys():
                print("%s values in [%s, %s] range, dim=%s" % (
                    key,
                    self.minv.get(key, 'N/A'),
                    self.maxv.get(key, 'N/A'),
                    self.jdim.get(key, 'N/A')))

def object_size(data):
    "Return size of the data"
    return sys.getsizeof(data.tobytes())

def size_format(uinput):
    """
    Format file size utility, it converts file size into KB, MB, GB, TB, PB units
    """
    try:
        num = float(uinput)
    except Exception as exc:
        print_exc(exc)
        return "N/A"
    base = 1000. # CMS convention to use power of 10
    if  base == 1000.: # power of 10
        xlist = ['', 'KB', 'MB', 'GB', 'TB', 'PB']
    elif base == 1024.: # power of 2
        xlist = ['', 'KiB', 'MiB', 'GiB', 'TiB', 'PiB']
    for xxx in xlist:
        if  num < base:
            return "%3.1f%s" % (num, xxx)
        num /= base

def parse(reader, nevts, verbose, fout, hists):
    "Parse given number of events from given reader"
    time0 = time.time()
    count = 0
    if nevts == -1:
        nevts = reader.nrows
    farr = []
    jarr = []
    for _ in range(nevts):
        xdf, _mask = reader.next(verbose=verbose)
        fdx = len(reader.flat_keys())
        flat = xdf[:fdx]
        jagged = xdf[fdx:]
        fsize = object_size(flat)
        jsize = object_size(jagged)
        farr.append(fsize)
        jarr.append(jsize)
        count += 1
    print("avg(flat)=%s, avg(jagged)=%s, ratio=%s" \
            % (size_format(np.mean(farr)), size_format(np.mean(jarr)), np.mean(farr)/np.mean(jarr)))
    totTime = time.time()-time0
    print("Read %s evts, %s Hz, total time %s" % (
        count, count/totTime, totTime))
    if fout:
        reader.write_specs(fout)
    if hg and hists:
        hgkeys = [k for k in reader.attrs]
        dump_histograms(reader.hdict, hgkeys)

def write(reader, nevts, verbose, fout):
    "Write given number of events from given reader to NumPy"
    time0 = time.time()
    count = 0
    with open(fout, 'wb+') as ostream:
        if nevts == -1:
            nevts = reader.nrows
        for _ in range(nevts):
            xdf = reader.next(verbose=verbose)
            ostream.write(xdf.tobytes())
            count += 1
        totTime = time.time()-time0
        print("Read %s evts, %s Hz, total time %s" % (
            count, count/totTime, totTime))

def xfile(fin, redirector='root://cms-xrd-global.cern.ch'):
    "Test if file is local or remote and setup proper prefix"
    if fin.startswith(redirector):
        return fin
    if os.path.exists(fin):
        return fin
    return "%s/%s" % (redirector, fin)

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    fin = opts.fin
    fout = opts.fout
    verbose = int(opts.verbose)
    nevts = int(opts.nevts)
    chunk_size = int(opts.chunk_size)
    nan = float(opts.nan)
    nevts = int(opts.nevts)
    specs = opts.specs
    branch = opts.branch
    branches = opts.branches.split(',') if opts.branches else []
    exclude_branches = []
    if opts.exclude_branches:
        if os.path.isfile(opts.exclude_branches):
            exclude_branches = \
                    [r.replace('\n', '') for r in open(opts.exclude_branches).readlines()]
        else:
            exclude_branches = opts.exclude_branches.split(',')
    hists = opts.hists
    identifier = [k.strip() for k in opts.identifier.split(',')]
    reader = DataReader(fin, branch=branch, selected_branches=branches,
            identifier=identifier, exclude_branches=exclude_branches, histograms=hists,
            nan=nan, chunk_size=chunk_size,
            nevts=nevts, specs=specs, redirector=opts.redirector, verbose=verbose)
    if opts.info:
        reader.info()
    else:
        parse(reader, nevts, verbose, fout, hists)

if __name__ == '__main__':
    main()
