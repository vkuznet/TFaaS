#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : tfaas.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: 
"""

# system modules
import os
import sys
import argparse
import traceback

# 3d part modules
import uproot
import pandas as pd

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input ROOT file")
        self.parser.add_argument("--branch", action="store",
            dest="branch", default="Events", help="Input ROOT file branch (default Events)")
        self.parser.add_argument("--branches", action="store",
            dest="branches", default="", help="ROOT branches to read, 'Electron_,Jet_'")
        self.parser.add_argument("--list-branches", action="store_true",
            dest="list_branches", default=False, help="list ROOT branches and exit")
        self.parser.add_argument("--fout", action="store",
            dest="fout", default="", help="Output model file")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")

def treeContent(tree):
    print(tree.contents)
    for key in tree.contents:
        branch = key.split(';')[0]
        print("key", key, branch)
        try:
            print(tree[branch])
        except:
            traceback.print_exc()

def listBranches(fin):
    "Executor function"
    try:
        with uproot.open(fin) as tree:
            print(tree)
            for key in tree.keys():
                branch = key.split(';')[0]
                eTree = tree[branch]
                print("### Branch {}".format(branch))
                for name in eTree.fBranches:
                    print(name)
    except:
        traceback.print_exc()

def run(fin, fout, branch, branches, verbose=False):
    "Executor function"
    try:
        with uproot.open(fin) as tree:
            eTree = tree[branch]
            print("Reading from branch: %s" % branch)
            for key, val in eTree.allitems():
                if key in branches:
                    print("Branch: %s" % key)
                    data = val.array()
                    print(data)
#             df = eTree.pandas.df(lambda b: b.dtype if b.name in branches else None)
#             print(df)
#             print(type(df))
    except:
        traceback.print_exc()


def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    if opts.list_branches:
        listBranches(opts.fin)
    else:
        branches = [b.strip() for b in opts.branches.split(',')]
        run(opts.fin, opts.fout, opts.branch, branches, opts.verbose)

if __name__ == '__main__':
    main()
