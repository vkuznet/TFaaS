// Example of using TFClient code
//
#include <thread>
#include <iostream>
#include <vector>
#include <sstream>
#include <sys/time.h>

#include "TFClient.h"

// add timestamp to benchmark our code
typedef unsigned long long timestamp_t;

static timestamp_t get_timestamp() {
  struct timeval now;
  gettimeofday (&now, NULL);
  return  now.tv_usec + (timestamp_t)now.tv_sec * 1000000;
}

// main function
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
        std::cout << "class: " << p.label() << " probability: " << p.probability() << std::endl;
    }

    // perform benchmark
    timestamp_t t0 = get_timestamp();
    auto ncalls = 100;
    for(int i=0; i<ncalls; i++) {
        auto res = predict(url, model, attrs, values);
    }
    timestamp_t t1 = get_timestamp();
    double secs = (t1 - t0) / 1000000.0L;
    std::cout << "single benchmark: " << ncalls << " calls in " << std::fixed << secs << " seconds" << std::endl;

    // threaded benchmark
    t0 = get_timestamp();
    std::vector<std::thread> threads;
    for(int i = 0; i < ncalls; ++i) {
        threads.push_back(std::thread(&predict, url, model, attrs, values));
    }
    for(auto& thread : threads){
        thread.join();
    }
    t1 = get_timestamp();
    secs = (t1 - t0) / 1000000.0L;
    std::cout << "thread benchmark: " << ncalls << " calls in " << std::fixed << secs << " seconds" << std::endl;
}
