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