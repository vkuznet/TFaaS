// -*- C++ -*-
//
// Package:    Demo/TFClient
// Class:      TFClient
// 
/**\class TFClient TFClient.cc Demo/TFClient/plugins/TFClient.cc

 Description: [one line class summary]

 Implementation:
     [Notes on implementation]
*/
//
// Original Author:  Valentin Y Kuznetsov
//         Created:  Wed, 03 Aug 2016 19:31:32 GMT
//
//
// Code examples used to write this code
// https://curl.haxx.se/libcurl/c/simple.html


// system include files
#include <memory>

// curl headers
#include <curl/curl.h>

// tfaas protobuf headers
#include "Demo/TFClient/interface/tfaas.pb.h"

#include <iostream>
using namespace std;

// helper function to read incoming stream of data, used as callback function
// in curl application below (see tfaasRequest)
static int writer(char *data, size_t size, size_t nmemb, string *writerData)
{
    if(writerData == NULL) return 0;
    writerData->append(data, size*nmemb);
    return size * nmemb;
}

// example from libcurl
// https://curl.haxx.se/libcurl/c/example.html
// https://curl.haxx.se/libcurl/c/htmltitle.html
// https://curl.haxx.se/libcurl/c/simplepost.html
// helper function to communicate with external data-service, in our case TFaaS
tfaaspb::Predictions tfaasRequest(const std::string& url, const std::string& input);
tfaaspb::Predictions tfaasRequest(const std::string& url, const std::string& input) {
    tfaaspb::Predictions preds;
    // read key/cert from environment
    auto ckey = getenv("X509_USER_PROXY");
    auto cert = getenv("X509_USER_PROXY");
    if (ckey == NULL || ckey == string("") || cert == NULL || cert == string("") ) {
        cerr << "Unable to read X509_USER_PROXY environment" << endl;
    }
    CURL *curl = NULL;
    CURLcode res;

    curl_global_init(CURL_GLOBAL_ALL);
    curl = curl_easy_init();
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 1L);
    curl_easy_setopt(curl, CURLOPT_SSLKEY, ckey);
    curl_easy_setopt(curl, CURLOPT_SSLCERT, cert);

    // Now specify the POST data
    curl_easy_setopt(curl, CURLOPT_POST, 1L);
    curl_easy_setopt(curl, CURLOPT_POSTFIELDS, input.c_str());
    curl_easy_setopt(curl, CURLOPT_POSTFIELDSIZE, input.length());

    string buffer;

    // send all data to this function
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, writer);
     
    // we pass our 'chunk' struct to the callback function
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, &buffer);
     
    // some servers don't like requests that are made without user-agent field, so we provide one
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "TFClient-libcurl-agent");

    res = curl_easy_perform(curl);
    if(res != CURLE_OK) {
        cout << "Failed to get: " << url << ", error " << curl_easy_strerror(res) << endl;
    } else {
        // read message from buffer
        // see examples on https://github.com/google/protobuf
        // https://developers.google.com/protocol-buffers/docs/cpptutorial
        GOOGLE_PROTOBUF_VERIFY_VERSION;
        if(!preds.ParseFromString(buffer)) {
            cerr << "failed to parse input buffer" << endl;
            return preds;
        }
        for (int i = 0; i < preds.prediction_size(); i++) {
            const tfaaspb::Class& cls = preds.prediction(i);
            cout << cls.label() << " probability: " << cls.probability() << endl;
        }
        google::protobuf::ShutdownProtobufLibrary();
    }
    // clean-up after we done with curl calls
    curl_easy_cleanup(curl);
    curl_global_cleanup();
    return preds;
}

tfaaspb::Predictions predict(const std::string& url, const std::string& model, const std::vector<std::string>& attrs, const std::vector<float>& values);
tfaaspb::Predictions predict(const std::string& url, const std::string& model, const std::vector<std::string>& attrs, const std::vector<float>& values)
{
    tfaaspb::Predictions preds;
    if(attrs.size() != values.size()) {
        cerr << "attributes size is not equal to values one" << endl;
        return preds;
    }

    // construct row message from given model/attributes/values
    tfaaspb::Row row;
    row.set_model(model);
    for(int i = 0; i < int(attrs.size()); i++) {
        row.add_key(attrs[i]);
        row.add_value(values[i]);
    }
    string input;
    if(!row.SerializeToString(&input)) {
        cerr << "unable to serialize data" << std::endl;
    } else {
        // send data to TFaaS and get back predictions for our data
        preds = tfaasRequest(url, input);
    }
    return preds;
}
