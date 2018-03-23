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
