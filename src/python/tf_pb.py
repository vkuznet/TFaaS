#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : tf_pb2pbtxt.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: This module takes care of TF model conversion from
binary (protobuf) to text (protobuf) representation. Code is based
on the following post:
https://stackoverflow.com/questions/38549153/how-to-convert-protobuf-graph-to-binary-wire-format#38706193
"""

# system modules
import os
import sys
import json
import argparse

# tensorflow
import tensorflow as tf
from tensorflow.python.platform import gfile
from google.protobuf import text_format, json_format

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input file")
        self.parser.add_argument("--fout", action="store",
            dest="fout", default="", help="Output file")
        self.parser.add_argument("--names", action="store_true",
            dest="names", default=False, help="Print out first/last node names of the graph")

def split(fname):
    "Helper function to split given file name into output dir and file name"
    arr = fname.split('/')
    if len(arr) == 1:
        fout = fname
        odir = os.getcwd()
    else:
        fout = arr[-1]
        odir = '/'.join(arr[:-1])
    pbdir = os.path.join(odir, '.pbdir')
    return odir, pbdir, fout

def convert_pb2pbtxt(fin, fout):
    "Convert given pb file into pbtxt one"
    odir, pbdir, fout = split(fout)
    print("Convert pb2pbtxt", odir, pbdir, fout)
    with gfile.FastGFile(fin,'rb') as f:
        graph_def = tf.GraphDef()
        graph_def.ParseFromString(f.read())
        tf.import_graph_def(graph_def, name='')
        tf.train.write_graph(graph_def, pbdir, fout, as_text=True)
        os.rename(os.path.join(pbdir, fout), os.path.join(odir, fout))
        os.rmdir(pbdir)
    return

def convert_pbtxt2pb(fin, fout):
    """Returns a `tf.GraphDef` proto representing the data in the given pbtxt file.

    Args:
    filename: The name of a file containing a GraphDef pbtxt (text-formatted
      `tf.GraphDef` protocol buffer data).

    Returns:
    A `tf.GraphDef` protocol buffer.
    """
    odir, pbdir, fout = split(fout)
    print("Convert pbtxt2pb", odir, pbdir, fout)
    with gfile.FastGFile(fin, 'r') as f:
        graph_def = tf.GraphDef()
        file_content = f.read()
        # Merges the human-readable string in `file_content` into `graph_def`.
        text_format.Merge(file_content, graph_def)
        tf.train.write_graph(graph_def, pbdir, fout, as_text=False)
        # move pb file
        os.rename(os.path.join(pbdir, fout), os.path.join(odir, fout))
        os.rmdir(pbdir)
    return

def convert(fin, fout):
    "Wrapper function"
    if fin.endswith('.pb'):
        convert_pb2pbtxt(fin, fout)
    elif fin.endswith('.pbtxt'):
        convert_pbtxt2pb(fin, fout)
    else:
        print("Ambiguous input file name, please specify it with .pb or .pbtxt extension")
        sys.exit(1)

def input_output(fin):
    "Parse given TF model file and return first/last node names"
    firstName = lastName = None
    with gfile.FastGFile(fin,'rb') as f:
        graph_def = tf.GraphDef()
        graph_def.ParseFromString(f.read())
        tf.import_graph_def(graph_def, name='')
        graph = json.loads(json_format.MessageToJson(graph_def))
        for key, nodes in graph.items():
            if key == 'node':
                for ndict in nodes:
                    if 'name' in ndict:
                        if not firstName:
                            firstName = ndict['name']
                        lastName = ndict['name']
        print firstName, lastName
    return firstName, lastName

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    if opts.names:
        input_output(opts.fin)
    else:
        convert(opts.fin, opts.fout)

if __name__ == '__main__':
    main()
