#include <iostream>

// tfaas protobuf headers
#include "tfaas.pb.h"

using namespace std;

tfaaspb::Predictions tfaasRequest(const std::string& url, const std::string& input);
tfaaspb::Predictions predict(const std::string& url, const std::string& model, const std::vector<std::string>& attrs, const std::vector<float>& values);
