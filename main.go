package main

import (
	"log"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

func main() {
	sdk, err := fabsdk.New(config.FromFile(""))
	defer sdk.Close()

	if err != nil {
		log.Fatalf("failed to create fabric sdk: %s", err)
	}

}
