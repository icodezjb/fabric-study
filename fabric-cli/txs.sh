#!/bin/bash
declare -i i=0
while [ $i -le $1 ];do
	./fabric-cli chaincode invoke --cid mychannel --ccid=mycc --args '{"Func":"invoke","Args":["a","b","1"]}' --peer grpcs://localhost:7051,grpcs://localhost:9051 --base64 --config ../config/org1sdk-config.yaml
	let i++
done
