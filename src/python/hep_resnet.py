#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : hep_resnet.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: HEP ResNet model
"""

from __future__ import print_function

# system modules
import os
import sys
import json
import time
import argparse

# numpy module
import numpy as np


# import keras module
try:
    import tensorflow
except ImportError:
    pass
from keras import backend as K
from keras.preprocessing.image import ImageDataGenerator
from keras.callbacks import ReduceLROnPlateau, CSVLogger, EarlyStopping
from keras.utils import np_utils

# import resnet model, see https://github.com/raghakot/keras-resnet
from resnet import ResnetBuilder

def get_img(fname):
    "Return numpy representation of the image of given file name"
    try:
        import cv2 # OpenCV
        return cv2.imread(fname)
    except ImportError:
        try:
            import imageio # python image module
            return imageio.imread(fname)
        except ImportError:
            from scipy import misc
            return misc.imread(fname) # scipy module

def get_model(model, ishape, nb_classes):
    "Return proper ResNet model"
    if model.endswith('18'):
        model = ResnetBuilder.build_resnet_18(ishape, nb_classes)
    elif model.endswith('34'):
        model = ResnetBuilder.build_resnet_34(ishape, nb_classes)
    elif model.endswith('50'):
        model = ResnetBuilder.build_resnet_50(ishape, nb_classes)
    elif model.endswith('101'):
        model = ResnetBuilder.build_resnet_101(ishape, nb_classes)
    elif model.endswith('152'):
        model = ResnetBuilder.build_resnet_152(ishape, nb_classes)
    else:
        return NotImplemented()
    return model

def save_tf_image(keras_model, odir, model_file='', graph_file='', quantize=False, num_output=1, output_node_prefix='output_node', backend='tf'):
    """
    Helper function to save TF model, see
    https://github.com/amir-abdi/keras_to_tensorflow/blob/master/keras_to_tensorflow.py

    quantize: if set to True, use the quantize feature of Tensorflow
    (https://www.tensorflow.org/performance/quantization)

    num_output: this value has nothing to do with the number of classes, batch_size, etc., 
    and it is mostly equal to 1. If the network is a **multi-stream network** 
    (forked network with multiple outputs), set the value to the number of outputs.

    output_node_prefix: the prefix to use for output nodes
    """
    from keras.models import load_model
    from keras import backend as K
    import tensorflow as tf

    if not os.path.isdir(odir):
        os.mkdir(odir)

    # Load keras model and rename output
    K.set_learning_phase(0)
    if backend == 'theano':
	K.set_image_data_format('channels_first')
    else:
	K.set_image_data_format('channels_last')

    # load keras model
    net_model = load_model(keras_model)

    if not model_file:
        model_file = 'model-%s.pb' % (time.strftime('%Y%m%d', time.gmtime()))

    pred = [None]*num_output
    pred_node_names = [None]*num_output
    for i in range(num_output):
	pred_node_names[i] = output_node_prefix+str(i)
        pred[i] = tf.identity(net_model.outputs[i], name=pred_node_names[i])
    print('output nodes names are: ', pred_node_names)

    # [optional] write graph definition in ascii
    sess = K.get_session()

    if graph_file:
	tf.train.write_graph(sess.graph.as_graph_def(), odir, graph_file, as_text=True)
	print('saved the graph definition in ascii format at: ', os.path.join(odir, graph_file))

    # convert variables to constants and save
    from tensorflow.python.framework import graph_util
    from tensorflow.python.framework import graph_io

    if quantize: # https://www.tensorflow.org/performance/quantization
        from tensorflow.tools.graph_transforms import TransformGraph
	transforms = ["quantize_weights", "quantize_nodes"]
	transformed_graph_def = TransformGraph(sess.graph.as_graph_def(), [], pred_node_names, transforms)
	constant_graph = graph_util.convert_variables_to_constants(sess, transformed_graph_def, pred_node_names)
    else:
	constant_graph = graph_util.convert_variables_to_constants(sess, sess.graph.as_graph_def(), pred_node_names)    
    graph_io.write_graph(constant_graph, odir, model_file, as_text=False)
    graph_io.write_graph(constant_graph, odir, model_file+'txt', as_text=True)
    print('saved TF model %s' % os.path.join(odir, model_file))

def load_data(fdir, flabel, ext='png', training=True):
    """Helper function to load HEP images. Images should be presented in the following
    directory structure:

    fdir: entry directory, e.g. images
          it should have train/test sub-folders and labels file
    flabel: labels file, e.g. labels.csv
        it lists image,label pairs, e.g. RelValQCD/file1,qcd
        image file name can point to other sub-directories and may or may not contain file extension

    Return array of image, their labels ids and number of classification classes
    """
    label_file = os.path.join(fdir, flabel)
    if os.path.isdir(os.path.join(fdir, 'train')):
        fdir = os.path.join(fdir, 'train')
    ldict = {}
    lnames = []
    fnames = []
    # fnames holds list of full paths to the images, e.g. /path/image1.png
    # lnames holds list of image labels
    # ldict holds mapping between image labels and ids
    with open(label_file, 'r') as istream:
        headers = istream.readline().replace('\n', '').split(',')
        while True:
            line = istream.readline().replace('\n', '')
            if not line:
                break
            arr = line.split(',')
            if len(arr) == 2:
                name, label = arr
            else: # no label
                name = arr[0]
                label = 'unknown'
            iname = os.path.join(fdir, name)
            if not iname.endswith(ext):
                iname = iname + '.' + ext
            fnames.append(iname)
            lnames.append(label)
            if label not in ldict:
                if ldict:
                    maxv = max(ldict.values())
                else:
                    maxv = -1
                ldict[label] = maxv + 1
    if len(ldict.values()) == 1 and ldict.values()[0] == 'unknown':
        labels_ids = [-1 for _ in lnames]
    else:
        labels_ids = [ldict[k] for k in lnames]
    print("image labels %s" % json.dumps(ldict))
    shape = None
    for idx, (fname, label) in enumerate(zip(fnames, labels_ids)):
        img = get_img(fname)
        if not shape:
            shape = np.shape(img)
            x_shape = [len(fnames)] + list(np.shape(img))
            x_data = np.empty(x_shape, dtype='uint8')
            y_data = np.empty((len(fnames),), dtype='uint8')
        x_data[idx:idx+1] = img
        y_data[idx:idx+1] = np.array([label])
    return x_data, y_data, len(labels_ids)

def run(params):
    "Business logic"
    input_shape = [int(k) for k in params.get('image_size', '300,300').split(',') if k]
    model_name = params.get('model', 'resnet18')
    optimizer = params.get('optimizer', 'adam')
    epochs = int(params.get('epochs', 100))
    batch_size = int(params.get('batch_size', 32))
    fdir = params.get('fdir', os.getcwd())
    flabels = params.get('flabels', 'labels.csv')
    augmentation = params.get('augmentation', False)
    ext = params.get('image_ext', 'png')
    split = float(params.get('split', 0.7))
    mdir = params.get('mdir', '')
    quantize = params.get('quantize', False)

    lr_reducer = ReduceLROnPlateau(factor=np.sqrt(0.1), cooldown=0, patience=5, min_lr=0.5e-6)
    early_stopper = EarlyStopping(min_delta=0.001, patience=10)
    csv_logger = CSVLogger('%s_hep.csv' % model_name)

    # load data and split it into train/validataion/test samples
    x_data, y_data, nb_classes = load_data(fdir, flabels, ext)
    y_data = np_utils.to_categorical(y_data, nb_classes)

    # split input DF into train/test sets
    sval = int(round(split*np.shape(x_data)[0])) # take split percentage value
    x_train, x_rest = np.vsplit(x_data, [sval]) # split into x_train and rest to x_rest
    y_train = y_data[:sval]
    y_rest = y_data[sval:]
    # split input test DF into valid/test sets
    sval = int(round(0.5*np.shape(x_rest)[0])) # take split percentage value
    x_valid, x_test = np.vsplit(x_rest, [sval]) # split into two equal sets
    y_valid = y_rest[:sval]
    y_test = y_rest[sval:]

    ishape = (3, input_shape[0], input_shape[1]) # RGB image NxM size
    model = get_model(model_name, ishape, nb_classes)

    # useful printouts
    print("Input set: %s shape" % str(np.shape(x_data)))
    print("Train set: %s shape" % str(np.shape(x_train)))
    print("Test  set: %s shape" % str(np.shape(x_test)))
    print("Valid set: %s shape" % str(np.shape(x_valid)))
    print("# classes: %s" % nb_classes)
    print("y_train  : %s" % str(np.shape(y_train)))
    print("y_test   : %s" % str(np.shape(y_test)))
    print("y_valid  : %s" % str(np.shape(y_valid)))
    print("model    : %s" % model)
    for layer in model.layers:
        print("layer    : %s, %s" % (layer.name, layer.input_spec))
    model.compile(loss='categorical_crossentropy', optimizer=optimizer, metrics=['accuracy'])
    if not augmentation:
	print('Not using data augmentation.')
        model.fit(x_train, y_train,
                  batch_size=batch_size,
                  epochs=epochs,
                  validation_data=(x_valid, y_valid),
                  shuffle=True,
                  callbacks=[lr_reducer, early_stopper, csv_logger])
    else:
	print('Using real-time data augmentation.')
	# This will do preprocessing and realtime data augmentation:
	datagen = ImageDataGenerator(
	    featurewise_center=False,  # set input mean to 0 over the dataset
	    samplewise_center=False,  # set each sample mean to 0
	    featurewise_std_normalization=False,  # divide inputs by std of the dataset
	    samplewise_std_normalization=False,  # divide each input by its std
	    zca_whitening=False,  # apply ZCA whitening
	    rotation_range=0,  # randomly rotate images in the range (degrees, 0 to 180)
	    width_shift_range=0.1,  # randomly shift images horizontally (fraction of total width)
	    height_shift_range=0.1,  # randomly shift images vertically (fraction of total height)
	    horizontal_flip=True,  # randomly flip images
	    vertical_flip=False)  # randomly flip images

	# Compute quantities required for featurewise normalization
	# (std, mean, and principal components if ZCA whitening is applied).
	datagen.fit(x_train)

	# Fit the model on the batches generated by datagen.flow().
	model.fit_generator(datagen.flow(x_train, y_train, batch_size=batch_size),
			    steps_per_epoch=x_train.shape[0] // batch_size,
			    validation_data=(x_valid, y_valid),
			    epochs=epochs, verbose=1, max_q_size=100,
			    callbacks=[lr_reducer, early_stopper, csv_logger])
    if mdir:
        try:
            os.makedirs(mdir)
        except:
            pass
        # save keras module
        tstamp = time.strftime('%Y%m%d', time.gmtime())
        kfile = os.path.join(mdir, 'keras_model_%s.h5' % tstamp)
        try:
            import h5py
            model.save(kfile)
            mfile = 'tf_model_%s.pb' % tstamp
            gfile = 'tf_graph_%s.pb' % tstamp
            save_tf_image(kfile, mdir, mfile, gfile, quantize)
        except ImportError:
            pass

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--fdir", action="store",
            dest="fdir", default="", help="Input directory")
        self.parser.add_argument("--flabels", action="store",
            dest="flabels", default="labels.csv", help="Image labels file name, default labels.csv")
        self.parser.add_argument("--model", action="store",
            dest="model", default="resnet18", help="model name, default resnet18")
        self.parser.add_argument("--mdir", action="store",
            dest="mdir", default="", help="Output directory for models")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")
        self.parser.add_argument("--augmentation", action="store_true",
            dest="augmentation", default=False, help="apply image augmentation")
        self.parser.add_argument("--quantize", action="store_true",
            dest="quantize", default=False, help="quantize TF model")
        self.parser.add_argument("--testing", action="store_true",
            dest="testing", default=False, help="Use model for testing")
        self.parser.add_argument("--batch_size", action="store",
            dest="batch_size", default=32, help="batch_size, default 32")
        self.parser.add_argument("--epochs", action="store",
            dest="epochs", default=100, help="epochs, default 100")
        self.parser.add_argument("--optimizer", action="store",
            dest="optimizer", default='adam', help="optimizer, default Adam")
	self.parser.add_argument('--image_size', type=str, default='300,300',
	    help='Image size to resize, default "300,300"')
	self.parser.add_argument('--image_ext', type=str, default='png',
	    help='Image extension, default png')
        self.parser.add_argument("--split", action="store",
            dest="split", default=0.7, help="split level for train sample, default 0.7")

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    params = {}
    params['fdir'] = opts.fdir
    params['flabels'] = opts.flabels
    params['verbose'] = opts.verbose
    params['model'] = opts.model
    params['image_size'] = opts.image_size
    params['image_ext'] = opts.image_ext
    params['optimizer'] = opts.optimizer
    params['epochs'] = opts.epochs
    params['batch_size'] = opts.batch_size
    params['augmentation'] = opts.augmentation 
    params['split'] = opts.split
    params['mdir'] = opts.mdir
    params['quantize'] = opts.quantize
    run(params)

if __name__ == '__main__':
    main()
