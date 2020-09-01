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

type Client struct {
	// Fabric network information
	cfg *Config

	// SDK Clients
	sdk *fabsdk.FabricSDK
	cc  *channel.Client
	lc  *ledger.Client

	//pack args function for chaincode calls
	packArgs func([]string) [][]byte
}

type SimpleClient interface {
	Send([]byte) error
}

type FabricClient interface {
	QueryBlockByNum(number uint64) (*common.Block, error)
	InvokeChainCode(fcn string, args []string) (fab.TransactionID, error)

	FilterEvents() []string
	Close()
}

func New(cfg *Config) *Client {
	c := &Client{
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

func (c *Client) initialize() {
	defer func() {
		if r := recover(); r != nil {
			utils.Fatalf("initialize fatal: %v", r)
		}
	}()

	c.initializeSDK()
	c.initializeChannelClient()
	c.initializeLedgerClient()
}

func (c *Client) initializeSDK() {
	sdk, err := fabsdk.New(c.cfg.ConfigProvider)
	if err != nil {
		utils.Fatalf("fabsdk.New err: %+v", err)
	}

	log.Info("Initialized fabric sdk")

	c.sdk = sdk
}

func (c *Client) initializeChannelClient() {
	channelProvider := c.sdk.ChannelContext(c.cfg.ChannelID(), fabsdk.WithUser(c.cfg.UserName()))

	cc, err := channel.New(channelProvider)
	if err != nil {
		utils.Fatalf("channel.New err: %v", err)
	}

	log.Info("Initialized channel client")

	c.cc = cc
}

func (c *Client) initializeLedgerClient() {
	channelProvider := c.sdk.ChannelContext(c.cfg.ChannelID(), fabsdk.WithUser(c.cfg.UserName()))
	lc, err := ledger.New(channelProvider)
	if err != nil {
		utils.Fatalf("ledger.New err: %v", err)
	}

	log.Info("Initialized ledger client")

	c.lc = lc
}

func (c *Client) QueryBlockByNum(number uint64) (*common.Block, error) {
	return c.lc.QueryBlock(number)
}

// InvokeChainCode("invoke", []string{"a", "b", "10"})
func (c *Client) InvokeChainCode(fcn string, args []string) (fab.TransactionID, error) {
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

func (c *Client) FilterEvents() []string {
	return c.cfg.FilterEvents
}

func (c *Client) Close() {
	c.sdk.Close()
}
