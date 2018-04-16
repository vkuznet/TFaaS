// Example of using TFClient code
//
#include <iostream>
#include <sstream>

#include "TFClient.h"

int main() {
    std::vector<std::string> attrs;
    std::vector<float> values;
    auto url = "http://localhost:8083/proto";
    auto model = "luca";

    // fill out our data
	for(int i=0; i<42; i++) {
        values.push_back(i);
        std::ostringstream oss;
        oss << i;
        attrs.push_back(oss.str());
    }

    // make prediction call
    auto res = predict(url, model, attrs, values);
    for(int i=0; i<res.prediction_size(); i++) {
        auto p = res.prediction(i);
        std::cout << "class: " << p.label() << " value: " << p.probability() << std::endl;
    }
}
