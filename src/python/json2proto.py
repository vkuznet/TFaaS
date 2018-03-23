#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : json2proto.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: convert given JSON file into protobuf one
"""

# system modules
import json
import argparse

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fin", action="store",
            dest="fin", default="", help="Input file")
        self.parser.add_argument("--fout", action="store",
            dest="fout", default="", help="Output file")

def convert(fin, fout):
    "Convert given JSON file into protobuf one"
    data = json.load(open(fin))
    with open(fout, 'w') as ostream:
        for key, val in zip(data['keys'], data['values']):
            ostream.write('key: "%s"\n' % key)
            ostream.write('value: %s\n' % val)

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    convert(opts.fin, opts.fout)

if __name__ == '__main__':
    main()
