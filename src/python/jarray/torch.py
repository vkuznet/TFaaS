#!/usr/bin/env python
#-*- coding: ISO-8859-1 -*-

"""
PyTorch implementation of Linear layer to deal with jagged array [1]
input data.

[1] https://www.wikiwand.com/en/Jagged_array
"""

from __future__ import print_function

__author__ = "Valentin Kuznetsov"

import numpy as np
import torch
from torch.autograd import Variable

class JaggedArrayLinear(torch.nn.modules.Linear):
    """
    JaggedArrayLinear is a wrapper around PyTorch Linear layer to deal
    with jagged array input data. The input data should be supplied as
    a numpy array of [data, mask] where data represents a data value
    vector and mask is associative mask for data vector. The mask vector
    should use np.nan for values which were added to data vector as paddings.
    """
    def __init__(self, in_features, out_features, bias=True, verbose=0):
        if verbose:
            print("### init JaggedArrayLinear, in {} out {}".format(in_features, out_features))
        super(JaggedArrayLinear, self).__init__(in_features, out_features, bias)

    def forward(self, data):
        "Linear layer implementation to deal with jagged array input data"
        # convert input data into torch tensor if necessary
        if isinstance(data, np.ndarray):
            data = Variable(torch.from_numpy(data).float())
        xdf, mask = data[0], data[1]
        # cast values in data vector according to the mask
        xdf[np.isnan(mask)] = 0
        return super(JaggedArrayLinear, self).forward(xdf)

def testJA():
    "Test code to test JaggedArrayLiner layer"
    model = torch.nn.Sequential(
        JaggedArrayLinear(10, 5),
        torch.nn.ReLU(),
        torch.nn.Linear(5, 1),
    )
    print(model)
    xdf = np.array(range(10))
    mask = np.ones(10)
    # assign some values in mask vector to nans
    mask[9] = mask[8] = np.nan
    data = np.array([xdf, mask])
    preds = model(data)
    print("preditctions {}".format(preds))

if __name__ == "__main__":
    testJA()
