### Installation
https://www.tensorflow.org/versions/master/install/install_go

go get github.com/tensorflow/tensorflow/tensorflow/go
go get github.com/tensorflow/tensorflow/tensorflow/go/op

### Model loading examples

We need to load TensorFlow model once in a while, I found the following post:
https://stackoverflow.com/questions/46427606/reload-tensorflow-model-in-golang-app-server

// Model Load Code:

    tags := []string{"serve"}

    // load from updated saved model
    var m *tensorflow.SavedModel
    var err error
    m, err = tensorflow.LoadSavedModel("/path/to/model", tags, nil)
    if err != nil {
        log.Errorf("Exception caught while reloading saved model %v", err)
        destroyTFModel(m)
    }

    if err == nil {
        ModelLoadMutex.Lock()
        defer ModelLoadMutex.Unlock()

        // destroy existing model
        destroyTFModel(TensorModel)
        TensorModel = m
    }


// Model Use Code(Part of the API request):

    config.ModelLoadMutex.RLock()
    defer config.ModelLoadMutex.RUnlock()

    scoreTensorList, err = TensorModel.Session.Run(map[tensorflow.Output]*tensorflow.Tensor{
        UserOp.Output(0): uT,
        DataOp.Output(0): nT},
        []tensorflow.Output{config.SumOp.Output(0)},
        nil,
    )

### References
another nice post:
https://outcrawl.com/image-recognition-api-go-tensorflow/

