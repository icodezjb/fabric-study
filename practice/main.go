package main

import (
	"time"

	"github.com/icodezjb/fabric-study/practice/client"
)

func main() {
	c := client.New("../config/org1sdk-config.yaml", "Org1", "Admin", "User1")
	defer c.Close()

	blkSync := client.NewBlockSync(c)
	defer blkSync.Stop()

	time.Sleep(60 * time.Minute)
}
