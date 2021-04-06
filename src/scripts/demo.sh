#!/bin/bash
echo "### obtain any existing ML model"
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --models"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --models
sleep 1
echo

echo "### upload new ML model"
echo
echo "cat upload.json"
echo
cat upload.json
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --upload=upload.json"
./tfaas_client.py --url=https://cms-tfaas.cern.ch --upload=upload.json
echo
echo "### upload tarball bundle"
tar tvfz /afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --bundle=/afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz"
./tfaas_client.py --url=https://cms-tfaas.cern.ch --bundle=/afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz
sleep 1
echo

echo "### view if our model exists"
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --models"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --models
sleep 2
echo

echo "### view if our model exists, but use jq tool to get better view over JSON"
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --models | jq"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --models | jq
sleep 2
echo

echo "### let's obtain some prediction"
echo
echo "cat input.json"
echo
cat input.json
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --predict=input.json"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --predict=input.json
sleep 2
echo

echo "### let's delete our ML model named vk"
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --delete=vk"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --delete=vk
./tfaas_client.py --url=https://cms-tfaas.cern.ch --delete=test_luca
sleep 1
echo

echo "### lets view again available models"
echo
echo "tfaas_client.py --url=https://cms-tfaas.cern.ch --models"
echo
./tfaas_client.py --url=https://cms-tfaas.cern.ch --models
sleep 2
echo

echo "### Let's repeat the same steps using curl client"
echo
echo "We define our curl client with some parameters and call it scurl"
echo "cat $HOME/bin/scurl"
echo
cat $HOME/bin/scurl
echo
sleep 2
echo

echo "### Let's view our models"
echo
echo "scurl -s https://cms-tfaas.cern.ch/models"
echo
scurl -s https://cms-tfaas.cern.ch/models
sleep 1
echo

echo "### let's send POST HTTP request with our parameters to upload ML model"
echo "### we provide params.json"
echo
cat params.json
echo
echo "### we provide model.pb TF model"
echo 
ls -al model.pb
echo
echo "### and we provide our labels in labels.txt file"
echo
cat labels.txt
echo
echo "### now we make scurl call"
echo
echo "scurl -s -X POST https://cms-tfaas.cern.ch/upload -F 'name=vk' -F 'params=@/afs/cern.ch/user/v/valya/workspace/models/vk/params.json' -F 'model=@/afs/cern.ch/user/v/valya/workspace/models/vk/model.pb' -F 'labels=@/afs/cern.ch/user/v/valya/workspace/models/vk/labels.txt'"
echo
scurl -s -X POST https://cms-tfaas.cern.ch/upload -F 'name=vk' -F 'params=@/afs/cern.ch/user/v/valya/workspace/models/vk/params.json' -F 'model=@/afs/cern.ch/user/v/valya/workspace/models/vk/model.pb' -F 'labels=@/afs/cern.ch/user/v/valya/workspace/models/vk/labels.txt'
sleep 1
echo

echo "### upload model bundle (saved from Keras TF)"
echo
echo "### content of a tarball"
tar tvfz /afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz
echo
sleep 1
echo "curl -v -X POST -H \"Content-Encoding: gzip\" -H \"content-type: application/octet-stream\" --data-binary @/afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz https://cms-tfaas.cern.ch/upload"
curl -v -X POST -H"Content-Encoding: gzip" -H"content-type: application/octet-stream " --data-binary @/afs/cern.ch/user/v/valya/workspace/test_luca.tar.gz https://cms-tfaas.cern.ch/upload
sleep 1
echo

echo "### Now we can view our models"
echo
echo "scurl -s https://cms-tfaas.cern.ch/models | jq"
echo
scurl -s https://cms-tfaas.cern.ch/models | jq
echo
sleep 2

echo "### And we can obtain our predictions using /json API"
echo
echo "scurl -s -X POST https://cms-tfaas.cern.ch/json -H "Content-type: application/json" -d@/afs/cern.ch/user/v/valya/workspace/models/vk/input.json"
echo
scurl -s -X POST https://cms-tfaas.cern.ch/json -H "Content-type: application/json" -d@/afs/cern.ch/user/v/valya/workspace/models/vk/input.json
sleep 1
echo

echo "### Now we can delete ML model using /delete end-point"
echo
echo "scurl -s -X DELETE https://cms-tfaas.cern.ch/delete -F 'model=vk'"
echo
scurl -s -X DELETE https://cms-tfaas.cern.ch/delete -F 'model=vk'
scurl -s -X DELETE https://cms-tfaas.cern.ch/delete -F 'model=test_luca'
sleep 1
echo

echo "### Now we can view our models"
echo
echo "scurl -s https://cms-tfaas.cern.ch/models"
echo
scurl -s https://cms-tfaas.cern.ch/models
echo
sleep 1

./tfaas_client.py --url=https://cms-tfaas.cern.ch --upload=upload.json
echo "### Now let's perform some stress tests"
echo "### for that we'll use hey tool which will send number of concurrent requests to tfaas service"
echo
echo "/cvmfs/cms.cern.ch/cmsmon/hey -m POST -H "Content-type: application/json" -D /afs/cern.ch/user/v/valya/workspace/models/vk/input.json https://cms-tfaas.cern.ch/json"
/cvmfs/cms.cern.ch/cmsmon/hey -m POST -H "Content-type: application/json" -D /afs/cern.ch/user/v/valya/workspace/models/vk/input.json https://cms-tfaas.cern.ch/json
