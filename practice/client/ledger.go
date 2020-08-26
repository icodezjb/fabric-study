package client

import (
	"fmt"
	"strconv"

	"github.com/icodezjb/fabric-study/practice/utils"
)

func (c *Client) QueryBlock(blockNum string) {
	intNum, _ := strconv.Atoi(blockNum)
	block, err := c.lc.QueryBlock(uint64(intNum))
	if err != nil {
		utils.Fatalf("QueryBlock err: %v", err)
	}

	preCrossTxs, err := GetPrepareCrossTxs(block, true)
	if err != nil {
		utils.Fatalf("ToFilteredBlock err: %v", err)
	}

	for _, preCrossTx := range preCrossTxs {
		fmt.Println(preCrossTx)
	}
}

func (c *Client) QueryChainInfo() {
	chainInfo, err := c.lc.QueryInfo()
	if err != nil {
		utils.Fatalf("QueryChainInfo err: %v", err)
	}

	fmt.Println(chainInfo)
}
