#### fabric test by fabric-go-sdk

- first-network: base on [hyperledger fabric-samples](https://github.com/hyperledger/fabric-samples) v1.4.2


#### start or stop byfn
```bash
cd first-network
./byfn.sh up
./byfn.sh down
./byfn.sh --help
```

#### fabric-cli
```bash
cd fabric-cli
go build
./fabric-cli chaincode query --cid mychannel --ccid mycc --args '{"Func":"query","Args":["a"]}' --peer grpcs://localhost:7051 --payload --config ../config/org1sdk-config.yaml
```

#### install your chaincode
```bash
cd first-network
./byfn.sh up -n
```

```bash
cd fabric-cli

########### install chaincode
./fabric-cli chaincode install --ccp=github.com/icodezjb/fabric-study/chaincode/chaincode_example02/go/ --ccid=mycc --v v1 --gopath `echo $GOPATH` --config ../config/org1sdk-config.yaml

########### upgrade
./fabric-cli chaincode install --ccp=github.com/icodezjb/fabric-study/chaincode/chaincode_example02/go/ --ccid=mycc --v v1 --gopath `echo $GOPATH` --config ../config/org1sdk-config.yaml
./fabric-cli chaincode upgrade --cid mychannel --ccp=github.com/icodezjb/fabric-study/chaincode/chaincode_example02/go/ --ccid=mycc --v v1 --args='{"Args":["init","a","100","b","100"]}' --policy "OutOf(2,'Org1MSP.member','Org2MSP.member')" --config ../config/org1sdk-config.yaml

########### instantiate
./fabric-cli chaincode instantiate --cid mychannel --ccp=github.com/icodezjb/fabric-study/chaincode/chaincode_example02/go/ --ccid=mycc --v v0 --args '{"Args":["init","a","100","b","100"]}' --policy "AND('Org1MSP.member','Org2MSP.member')" --config ../config/org1sdk-config.yaml

########### invoke
./fabric-cli chaincode invoke --cid mychannel --ccid=mycc --args '{"Func":"invoke","Args":["a","b","1"]}' --peer grpcs://localhost:7051,grpcs://localhost:9051 --base64 --config ../config/org1sdk-config.yam

########### query
./fabric-cli query info --cid mychannel  --config ../config/org1sdk-config.yaml 

./fabric-cli query block --cid mychannel --num 13 --base64 --config ../config/org1sdk-config.yaml 

./fabric-cli query block --cid mychannel --hash n2AMgN_gHCCvPBLZvKFGOYKcWO4iO8r1339w4s45rmQ --base64 --config ../config/org1sdk-config.yaml

./fabric-cli query block --cid mychannel --hash n2AMgN_gHCCvPBLZvKFGOYKcWO4iO8r1339w4s45rmQ --base64 --format json --config ../config/org1sdk-config.yaml

./fabric-cli query channels --peer grpcs://localhost:7051 --config ../config/org1sdk-config.yaml

./fabric-cli query tx --cid mychannel --txid d95018d5f9d3a83e6734e7c3390e64723eefbcc07633fc81a9fd950a31eebadc --base64 --config ../config/org1sdk-config.yaml

./fabric-cli chaincode query --cid mychannel --ccid mycc --payload --args '{"Func":"query","Args":["b"]}' --peer grpcs://localhost:7051,grpcs://localhost:8051 --config ../config/org1sdk-config.yaml 

./fabric-cli chaincode query --cid mychannel --ccid mycc --args '{"Func":"query","Args":["a"]}' --peer grpcs://localhost:7051 --payload --config ../config/org1sdk-config.yaml

./fabric-cli query installed --peer localhost:7051 --config ../config/org1sdk-config.yaml


########### event
./fabric-cli event listenfilteredblock --cid mychannel --base64 --config ../config/org1sdk-config.yaml

./fabric-cli event listencc --cid mychannel --ccid=mycc --event=.* --config ../config/org1sdk-config.yaml
```

