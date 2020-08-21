package main

import (
	"log"

	"github.com/icodezjb/fabric-study/client"
)

func main() {
	c := client.New("./config/org1sdk-config.yaml", "Org1", "Admin", "User1")
	defer c.Close()

	if err := c.QueryChainCode("peer0.org1.example.com", "a"); err != nil {
		log.Fatalln("Query chaincode error: %v", err)
	}

	log.Println("Query chaincode success on peer0.org1")

	log.Println("Query block 1 ")

	c.QueryBlock(1)
}
