package main

import (
	"fmt"
	"github.com/icodezjb/fabric-study/practice/client"

	"github.com/icodezjb/fabric-study/practice/utils"
)

func main() {
	c := client.New("./config/org1sdk-config.yaml", "Org1", "Admin", "User1")
	defer c.Close()

	if err := c.QueryChainCode("peer0.org1.example.com", "a"); err != nil {
		utils.Fatalf("Query chaincode error: %v", err)
	}

	fmt.Println("Query chaincode success on peer0.org1")

	fmt.Println("Query block 1 ")

	c.QueryBlock(1)
}
