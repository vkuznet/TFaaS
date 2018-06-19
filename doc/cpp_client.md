#### C++ client
Here we present only code how to make inference call to TFaaS server:
```
#include <iostream>
#include <vector>
#include <sstream>
#include “TFClient.h”                              // include TFClient header

// main function
int main() {
    std::vector<std::string> attrs;                // define vector of attributes
    std::vector<float> values;                     // define vector of values
    auto url = “http://localhost:8083/proto”;      // define your TFaaS URL
    auto model = “MyModel";                        // name your model

    // fill out our data
    for(int i=0; i<42; i++) {                      // the model I tested had 42 parameters
        values.push_back(i);                       // create your vector values
        std::ostringstream oss;
        oss << i;
        attrs.push_back(oss.str());                // create your vector headers
    }

    // make prediction call
    auto res = predict(url, model, attrs, values); // get predictions from TFaaS
    for(int i=0; i<res.prediction_size(); i++) {
        auto p = res.prediction(i);                // fetch and print model predictions
        std::cout << "class: " << p.label() << " probability: " << p.probability() << std::endl;
    }
}
```


