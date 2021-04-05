#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : tfaas_client.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: simple python client to communicate with TFaaS server
"""

# system modules
import os
import sys
import pwd
import ssl
import json
import binascii
import argparse
import itertools
import mimetools
import mimetypes
if  sys.version_info < (2, 7):
    raise Exception("TFaaS client requires python 2.7 or greater")
# python 3
if  sys.version.startswith('3.'):
    import urllib.request as urllib2
    import urllib.parse as urllib
    import http.client as httplib
    import http.cookiejar as cookielib
else:
    import urllib
    import urllib2
    import httplib
    import cookielib

TFAAS_CLIENT = 'tfaas-client/1.1::python/%s.%s' % sys.version_info[:2]

class OptionParser():
    def __init__(self):
        "User based option parser"
        self.parser = argparse.ArgumentParser(prog='PROG')
        self.parser.add_argument("--url", action="store",
            dest="url", default="", help="TFaaS URL")
        self.parser.add_argument("--upload", action="store",
            dest="upload", default="", help="upload model to TFaaS")
        self.parser.add_argument("--bundle", action="store",
            dest="bundle", default="", help="upload bundle ML files to TFaaS")
        self.parser.add_argument("--predict", action="store",
            dest="predict", default="", help="fetch prediction from TFaaS")
        self.parser.add_argument("--image", action="store",
            dest="image", default="", help="fetch prediction for given image")
        self.parser.add_argument("--model", action="store",
            dest="model", default="", help="TF model to use")
        self.parser.add_argument("--delete", action="store",
            dest="delete", default="", help="delete model in TFaaS")
        self.parser.add_argument("--models", action="store_true",
            dest="models", default=False, help="show existing models in TFaaS")
        self.parser.add_argument("--verbose", action="store_true",
            dest="verbose", default=False, help="verbose output")
        msg  = 'specify private key file name, default $X509_USER_PROXY'
        self.parser.add_argument("--key", action="store",
                               default=x509(), dest="ckey", help=msg)
        msg  = 'specify private certificate file name, default $X509_USER_PROXY'
        self.parser.add_argument("--cert", action="store",
                               default=x509(), dest="cert", help=msg)
        default_ca = os.environ.get("X509_CERT_DIR")
        if not default_ca or not os.path.exists(default_ca):
            default_ca = "/etc/grid-security/certificates"
            if not os.path.exists(default_ca):
                default_ca = ""
        if default_ca:
            msg = 'specify CA path, default currently is %s' % default_ca
        else:
            msg = 'specify CA path; defaults to system CAs.'
        self.parser.add_argument("--capath", action="store",
                               default=default_ca, dest="capath", help=msg)
        msg  = 'specify number of retries upon busy DAS server message'

class HTTPSClientAuthHandler(urllib2.HTTPSHandler):
    """
    Simple HTTPS client authentication class based on provided
    key/ca information
    """
    def __init__(self, key=None, cert=None, capath=None, level=0):
        if  level > 0:
            urllib2.HTTPSHandler.__init__(self, debuglevel=1)
        else:
            urllib2.HTTPSHandler.__init__(self)
        self.key = key
        self.cert = cert
        self.capath = capath

    def https_open(self, req):
        """Open request method"""
        #Rather than pass in a reference to a connection class, we pass in
        # a reference to a function which, for all intents and purposes,
        # will behave as a constructor
        return self.do_open(self.get_connection, req)

    def get_connection(self, host, timeout=300):
        """Connection method"""
        if  self.key and self.cert and not self.capath:
            return httplib.HTTPSConnection(host, key_file=self.key,
                                                cert_file=self.cert)
        elif self.cert and self.capath:
            context = ssl.SSLContext(ssl.PROTOCOL_TLSv1)
            context.load_verify_locations(capath=self.capath)
            context.load_cert_chain(self.cert)
            return httplib.HTTPSConnection(host, context=context)
        return httplib.HTTPSConnection(host)

def x509():
    "Helper function to get x509 either from env or tmp file"
    proxy = os.environ.get('X509_USER_PROXY', '')
    if  not proxy:
        proxy = '/tmp/x509up_u%s' % pwd.getpwuid( os.getuid() ).pw_uid
        if  not os.path.isfile(proxy):
            return ''
    return proxy

def check_auth(key):
    "Check if user runs das_client with key/cert and warn users to switch"
    if  not key:
        msg  = "WARNING: tfaas_client is running without user credentials/X509 proxy, create proxy via 'voms-proxy-init -voms cms -rfc'"
        print(msg)

def fullpath(path):
    "Expand path to full path"
    if  path and path[0] == '~':
        path = path.replace('~', '')
        path = path[1:] if path[0] == '/' else path
        path = os.path.join(os.environ['HOME'], path)
    return path

# credit: https://pymotw.com/2/urllib2/#uploading-files
class MultiPartForm(object):
    """Accumulate the data to be used when posting a form."""

    def __init__(self):
        self.form_fields = []
        self.files = []
        self.boundary = mimetools.choose_boundary()
        return
    
    def get_content_type(self):
        return 'multipart/form-data; boundary=%s' % self.boundary

    def add_field(self, name, value):
        """Add a simple field to the form data."""
        self.form_fields.append((name, value))
        return

    def add_file(self, fieldname, filename, fileHandle, mimetype=None):
        """Add a file to be uploaded."""
        body = fileHandle.read()
        if mimetype is None:
            mimetype = mimetypes.guess_type(filename)[0] or 'application/octet-stream'
        if mimetype == 'application/octet-stream':
            body = binascii.b2a_base64(body)
        self.files.append((fieldname, filename, mimetype, body))
        return
    
    def __str__(self):
        """Return a string representing the form data, including attached files."""
        # Build a list of lists, each containing "lines" of the
        # request.  Each part is separated by a boundary string.
        # Once the list is built, return a string where each
        # line is separated by '\r\n'.  
        parts = []
        part_boundary = '--' + self.boundary
        
        # Add the form fields
        parts.extend(
            [ part_boundary,
              'Content-Disposition: form-data; name="%s"' % name,
              '',
              value,
            ]
            for name, value in self.form_fields
            )
        
        # Add the files to upload
        # here we use form-data content disposition instead of file one
        # since this is how we define handlers in our Go server
        # for more info see: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Disposition
        parts.extend(
            [ part_boundary,
              'Content-Disposition: form-data; name="%s"; filename="%s"' % \
                 (field_name, filename),
              'Content-Type: %s' % content_type,
              '',
              body,
            ]
            for field_name, filename, content_type, body in self.files
            )
        
        # Flatten the list and add closing boundary marker,
        # then return CR+LF separated data
        flattened = list(itertools.chain(*parts))
        flattened.append('--' + self.boundary + '--')
        flattened.append('')
        return '\r\n'.join(flattened)

def models(host, verbose=None, ckey=None, cert=None, capath=None):
    "models API shows models from TFaaS server"
    url = host + '/models'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"Accept": "application/json", "User-Agent": client}
    if verbose:
        print("URL   : %s" % url)
    encoded_data = json.dumps({})
    return getdata(url, headers, encoded_data, ckey, cert, capath, verbose, 'GET')

def delete(host, model, verbose=None, ckey=None, cert=None, capath=None):
    "delete API deletes given model in TFaaS server"
    url = host + '/delete'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"User-Agent": client}
    if verbose:
        print("URL   : %s" % url)
        print("model : %s" % model)
    form = MultiPartForm()
    form.add_field('model', model)
    edata = str(form)
    headers['Content-length'] = len(edata)
    headers['Content-Type'] = form.get_content_type()
    return getdata(url, headers, edata, ckey, cert, capath, verbose, method='DELETE')

def bundle(host, ifile, verbose=None, ckey=None, cert=None, capath=None):
    "bundle API uploads given bundle model files to TFaaS server"
    url = host + '/upload'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"User-Agent": client, "Content-Encoding": "gzip", "Content-Type": "application/octet-stream"}
    data = open(ifile, 'rb').read()
    return getdata(url, headers, data, ckey, cert, capath, verbose)

def upload(host, ifile, verbose=None, ckey=None, cert=None, capath=None):
    "upload API uploads given model to TFaaS server"
    url = host + '/upload'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"User-Agent": client}
    params = json.load(open(ifile))
    if verbose:
        print("URL   : %s" % url)
        print("ifile : %s" % ifile)
        print("params: %s" % json.dumps(params))

    form = MultiPartForm()
    for key in params.keys():
        if key in ['model', 'labels', 'params']:
            flag = 'r'
            if key == 'model':
                flag = 'rb'
            name = params[key]
            form.add_file(key, name, fileHandle=open(name, flag))
        else:
            form.add_field(key, params[key])
    edata = str(form)
    headers['Content-length'] = len(edata)
    headers['Content-Type'] = form.get_content_type()
    headers['Content-Encoding'] = 'base64'
    return getdata(url, headers, edata, ckey, cert, capath, verbose)

def predict(host, ifile, model, verbose=None, ckey=None, cert=None, capath=None):
    "predict API get predictions from TFaaS server"
    url = host + '/json'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"Accept": "application/json", "User-Agent": client}
    params = json.load(open(ifile))
    if model: # overwrite model name in given input file
        params['model'] = model
    if verbose:
        print("URL   : %s" % url)
        print("ifile : %s" % ifile)
        print("params: %s" % json.dumps(params))
    encoded_data = json.dumps(params)
    return getdata(url, headers, encoded_data, ckey, cert, capath, verbose)

def predictImage(host, ifile, model, verbose=None, ckey=None, cert=None, capath=None):
    "predict API get predictions from TFaaS server"
    url = host + '/image'
    client = '%s (%s)' % (TFAAS_CLIENT, os.environ.get('USER', ''))
    headers = {"Accept": "application/json", "User-Agent": client}
    if verbose:
        print("URL   : %s" % url)
        print("ifile : %s" % ifile)
        print("model : %s" % model)
    form = MultiPartForm()
    form.add_file('image', ifile, fileHandle=open(ifile, 'r'))
    form.add_field('model', model)
    edata = str(form)
    headers['Content-length'] = len(edata)
    headers['Content-Type'] = form.get_content_type()
    return getdata(url, headers, edata, ckey, cert, capath, verbose)

def getdata(url, headers, encoded_data, ckey, cert, capath, verbose=None, method='POST'):
    "helper function to use in predict/upload APIs, it place given URL call to the server"
    debug = 1 if verbose else 0
    req = urllib2.Request(url=url, headers=headers, data=encoded_data)
    if method == 'DELETE':
        req.get_method = lambda: 'DELETE'
    elif method == 'GET':
        req = urllib2.Request(url=url, headers=headers)
    if  ckey and cert:
        ckey = fullpath(ckey)
        cert = fullpath(cert)
        http_hdlr  = HTTPSClientAuthHandler(ckey, cert, capath, debug)
    elif cert and capath:
        cert = fullpath(cert)
        http_hdlr  = HTTPSClientAuthHandler(ckey, cert, capath, debug)
    else:
        http_hdlr  = urllib2.HTTPHandler(debuglevel=debug)
    proxy_handler  = urllib2.ProxyHandler({})
    cookie_jar     = cookielib.CookieJar()
    cookie_handler = urllib2.HTTPCookieProcessor(cookie_jar)
    data = {}
    try:
        opener = urllib2.build_opener(http_hdlr, proxy_handler, cookie_handler)
        fdesc = opener.open(req)
        if url.endswith('json'):
            data = json.load(fdesc)
        else:
            data = fdesc.read()
        fdesc.close()
    except urllib2.HTTPError as error:
        print(error.read())
        sys.exit(1)
    if url.endswith('json'):
        return json.dumps(data)
    return data

def main():
    "Main function"
    optmgr  = OptionParser()
    opts = optmgr.parser.parse_args()
    check_auth(opts.ckey)
    res = ''
    if opts.upload:
        res = upload(opts.url, opts.upload, opts.verbose, opts.ckey, opts.cert, opts.capath)
    if opts.bundle:
        res = bundle(opts.url, opts.bundle, opts.verbose, opts.ckey, opts.cert, opts.capath)
    elif opts.delete:
        res = delete(opts.url, opts.delete, opts.verbose, opts.ckey, opts.cert, opts.capath)
    elif opts.models:
        res = models(opts.url, opts.verbose, opts.ckey, opts.cert, opts.capath)
    elif opts.predict:
        res = predict(opts.url, opts.predict, opts.model, opts.verbose, opts.ckey, opts.cert, opts.capath)
    elif opts.image:
        res = predictImage(opts.url, opts.image, opts.model, opts.verbose, opts.ckey, opts.cert, opts.capath)
    if res:
        print(res)

if __name__ == '__main__':
    main()
