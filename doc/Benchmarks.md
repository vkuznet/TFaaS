### TFaaS benchmark
We teste TFaaS server on the following node: CentOS 7 Linux, 16 cores, 30GB of RAM.
The benchmarks was done using modified version of
[hey](https://github.com/vkuznet/hey) tool using the following command:
```
# in this example we used 1000 calls with 100 concurrent clients
# the input.json file contains set of input parameters we sent to TFaaS
hey -n 1000 -c 100 -m POST -H "Content-type: application/json" -D input.json https://localhost:8083/json
```

The results are the following:
- 1000 calls with 100 concurrent clients:
```
Summary:
  Total:        2.4778 secs
  Slowest:      1.3385 secs
  Fastest:      0.0045 secs
  Average:      0.2135 secs
  Requests/sec: 403.5864
  Total data:   12000 bytes
  Size/request: 12 bytes

Status code distribution:
  [200] 1000 responses

Response time histogram:
  0.005 [1]     |
  0.138 [508]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.271 [382]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.405 [10]    |∎
  0.538 [5]     |
  0.672 [5]     |
  0.805 [8]     |∎
  0.938 [11]    |∎
  1.072 [14]    |∎
  1.205 [27]    |∎∎
  1.339 [29]    |∎∎
```

- 5000 calls with 200 concurrent clients:
```
Summary:
  Total:        10.5098 secs
  Slowest:      2.9215 secs
  Fastest:      0.0051 secs
  Average:      0.3828 secs
  Requests/sec: 475.7460
  Total data:   60000 bytes
  Size/request: 12 bytes

Status code distribution:
  [200] 5000 responses

Response time histogram:
  0.005 [1]     |
  0.297 [2079]  |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.588 [2603]  |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.880 [132]   |∎∎
  1.172 [5]     |
  1.463 [12]    |
  1.755 [20]    |
  2.047 [28]    |
  2.338 [42]    |∎
  2.630 [74]    |∎
  2.921 [4]     |
```

For protobuffer data we used the following commands:
```
# first we create a proto.msg file with our input data
# key: "attr1"
# value: 0.1
# ....
#
# then we converted this message into binary form, here /path should point to
# location of proto area which contains our tfaaspb definitions
cat proto.msg | protoc -I/path/proto --encode tfaaspb.Row tfaaspb.Predictions > proto.bin

# call hey with the following options 
# 1000 calls with 100 concurrent clients
hey -n 1000 -c 100 -m POST -D proto.bin https://localhost:8083/proto
# 5000 calls with 200 concurrent clients
hey -n 5000 -c 200 -m POST -D proto.bin https://localhost:8083/proto
```

The results are the following:
- 1000 calls with 100 concurrent clients:
```
Summary:
  Total:        2.5362 secs
  Slowest:      1.4648 secs
  Fastest:      0.0050 secs
  Average:      0.2194 secs
  Requests/sec: 394.2874
  Total data:   7000 bytes
  Size/request: 7 bytes

Status code distribution:
  [200] 1000 responses

Response time histogram:
  0.005 [1]     |
  0.151 [573]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.297 [322]   |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.443 [9]     |∎
  0.589 [1]     |
  0.735 [4]     |
  0.881 [10]    |∎
  1.027 [18]    |∎
  1.173 [15]    |∎
  1.319 [41]    |∎∎∎
  1.465 [6]     |
```

- 5000 calls with 200 concurrent clients:
```
Summary:
  Total:        10.3902 secs
  Slowest:      2.7922 secs
  Fastest:      0.0044 secs
  Average:      0.3775 secs
  Requests/sec: 481.2210
  Total data:   35000 bytes
  Size/request: 7 bytes

Status code distribution:
  [200] 5000 responses

Response time histogram:
  0.004 [1]     |
  0.283 [1985]  |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.562 [2664]  |∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
  0.841 [162]   |∎∎
  1.120 [7]     |
  1.398 [4]     |
  1.677 [15]    |
  1.956 [22]    |
  2.235 [37]    |∎
  2.513 [78]    |∎
  2.792 [25]    |
```

As you can see from these tests the results between JSON and proto exchange with
TFaaS server are quite similar.

