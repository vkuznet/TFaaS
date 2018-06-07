### TensorFlow as a Service (TFaaS)

A general purpose framework to serve TensorFlow models.
It provides reach and flexible set of APIs to efficiently manage your
favorite TF models. The TFaaS supports JSON and ProtoBuffer data-formats.

The following set of APIs is provided:
- */upload* to push your favorite TF model to TFaaS server
- */delete* to delete your TF model from TFaaS server
- */models* to view existing TF models on TFaaS server
- */json* and */proto* to serve TF models predictions in corresponding
  data-format

#### TFaaS deployment
Install TFaaS server via docker image:
```
docker run --rm -h `hostname -f` -p 8083:8083 -i -t veknet/tfaas
```

Prepare parameters to upload your model
```
{
  "name": "ImageModel", "model": "tf_model.pb", "labels": "labels.txt",
  "inputNode": "dense_4_input", "outputNode": "output_node0"
}
```
Upload your favorite model (we name it as *ImageModel*)
```
curl -X POST http://localhost:8083/upload -F 'name=ImageModel'
-F 'params=@/path/params.json'
-F 'model=@/path/tf_model.pb' -F 'labels=@/path/labels.txt'
```
Get predictions:
```
curl https://localhost:8083/image -F 'image=@/path/file.png' -F 'model=ImageModel'
```

#### TFaaS benchmarks
Benchmark results on CentOS, 24 cores, 32GB of RAM serving DL NN with
42x128x128x128x64x64x1x1 architecture:
- 400 req/sec for 100 concurrent clients, 1000 requests in total
- 480 req/sec for 200 concurrent clients, 5000 requests in total
JSON and ProtoBuffer formats show similar performance.
