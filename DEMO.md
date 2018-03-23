### TFaaS demo
The TFaaS service is easy to run for that we need few steps:
- create `/tmp/tfaas` area where we store secrets
- obtain your proxy via `voms-proxy-init -voms cms -rfc` command and copy it into /tmp/tfaas/tfaas-proxy
- optionally, obtain `server.crt` and `server.key` files and put them into /tmp/tfaas area

To run the service you can invoke as following:
```
docker run --rm -h `hostname -f` -p 8083:8083 -v /tmp/tfaas:/etc/secrets -i -t veknet/tfaas
```

First, it will download the image, then it will start the tfaas server on your
machine. It mounts `/tmp/tfaas` directory under `/etc/secrets` inside the container
such that `tfaas-proxy`, `server.*` files will be available for tfaas. And, it
open 8083 port on your local host to access the server.

NOTE: uploading `server.*` files is optional, if you do that tfaas will start
as HTTPs server, if you do not place them it will start as HTTP server.
Depending on this you'll need to adjust #6 step below to use either http
or https call.

Finally, we can:
- upload the model and label files
- change tfaas parameters
- and fire up HTTP cal to get your predictions.

For the demo I used the following:

1. check existing models in tfaas

   `scurl -i https://localhost:8083/models`

2. upload TF model and labels into tfaas

   `scurl -i -X POST https://localhost:8083/upload -F 'model=@model_0228.pb' -F 'labels=@labels.txt'`

3. check existing model parameters

   `scurl -i https://localhost:8083/params`

4. create params.json with your parameters and upload it to tfaas server

   `scurl -i -X POST -H "Content-type: application/json" -d @params.json https://localhost:8083/params`

5. do #3 again just to be sure that your params will fit with your model (inputNode, outputNode, etc.)

6. place your HTTP call e.g. via curl

   `scurl -H "Content-type: application/json" -d '{"key":[...], "values":[...]}' https://localhost:8083/json`

Full demo can be found
[here](https://www.youtube.com/watch?v=ZGjnM8wk8eA)

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
