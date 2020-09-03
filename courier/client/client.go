package client

import (
	"github.com/icodezjb/fabric-study/courier/utils"
	"github.com/icodezjb/fabric-study/log"

	"github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type FClient struct {
	// Fabric network information
	cfg *Config

	// SDK Clients
	sdk *fabsdk.FabricSDK
	cc  *channel.Client
	lc  *ledger.Client

	//pack args function for chaincode calls
	packArgs func([]string) [][]byte
}

type OutChainClient interface {
	Send([]byte) error
	Close()
}

type MockOutChainClient struct {
	count uint32
}

func (mc *MockOutChainClient) Send([]byte) error {
	mc.count++
	log.Info("send to OutChain", "count", mc.count)
	return nil
}

func (mc *MockOutChainClient) Close() {

}

type FabricClient interface {
	QueryBlockByNum(number uint64) (*common.Block, error)
	InvokeChainCode(fcn string, args []string) (fab.TransactionID, error)

	FilterEvents() []string
	Close()
}

func NewFabCli(cfg *Config) *FClient {
	c := &FClient{
		cfg: cfg,

		packArgs: func(params []string) [][]byte {
			var args [][]byte
			for _, param := range params {
				args = append(args, []byte(param))
			}
			return args
		},
	}

	c.initialize()

	return c
}

func (c *FClient) initialize() {
	defer func() {
		if r := recover(); r != nil {
			utils.Fatalf("[FClient] initialize fatal: %v", r)
		}
	}()

	c.initializeSDK()
	c.initializeChannelClient()
	c.initializeLedgerClient()
}

func (c *FClient) initializeSDK() {
	sdk, err := fabsdk.New(c.cfg.ConfigProvider)
	if err != nil {
		utils.Fatalf("[FClient] fabsdk.New err: %+v", err)
	}

	log.Info("[FClient] initialized fabric sdk")

	c.sdk = sdk
}

func (c *FClient) initializeChannelClient() {
	channelProvider := c.sdk.ChannelContext(c.cfg.ChannelID(), fabsdk.WithUser(c.cfg.UserName()))

	cc, err := channel.New(channelProvider)
	if err != nil {
		utils.Fatalf("[FClient] channel.New err: %v", err)
	}

	log.Info("[FClient] initialized channel client")

	c.cc = cc
}

func (c *FClient) initializeLedgerClient() {
	channelProvider := c.sdk.ChannelContext(c.cfg.ChannelID(), fabsdk.WithUser(c.cfg.UserName()))
	lc, err := ledger.New(channelProvider)
	if err != nil {
		utils.Fatalf("[FClient] ledger.New err: %v", err)
	}

	log.Info("[FClient] initialized ledger client")

	c.lc = lc
}

func (c *FClient) QueryBlockByNum(number uint64) (*common.Block, error) {
	return c.lc.QueryBlock(number)
}

// InvokeChainCode("invoke", []string{"a", "b", "10"})
func (c *FClient) InvokeChainCode(fcn string, args []string) (fab.TransactionID, error) {
	req := channel.Request{
		ChaincodeID: c.cfg.ChainCodeID(),
		Fcn:         fcn,
		Args:        c.packArgs(args),
	}
	resp, err := c.cc.Execute(req, c.cfg.RequestOption)
	if err != nil {
		return "", err
	}

	return resp.TransactionID, nil
}

func (c *FClient) FilterEvents() []string {
	return c.cfg.FilterEvents
}

func (c *FClient) Close() {
	c.sdk.Close()
}
