package client

import (
	"fmt"
	"os"

	"github.com/icodezjb/fabric-study/practice/utils"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

type Client struct {
	// Fabric network information
	ConfigPath string
	OrgName    string
	OrgAdmin   string
	OrgUser    string

	// SDK Clients
	SDK *fabsdk.FabricSDK
	rc  *resmgmt.Client
	cc  *channel.Client
	lc  *ledger.Client

	ChannelID       string
	ChainCodeID     string
	ChainCodePath   string // chaincode source path, in GOPATH
	ChainCodeGoPath string // GOPATH

	//pack args function for chaincode calls
	PackArgs func([]string) [][]byte
}

func New(cfg, orgName, orgAdmin, orgUser string) *Client {
	c := &Client{
		ConfigPath: cfg,
		OrgName:    orgName,
		OrgAdmin:   orgAdmin,
		OrgUser:    orgUser,

		ChannelID:       "mychannel",
		ChainCodeID:     "mycc",
		ChainCodePath:   "github.com/icodezjb/fabric-study/chaincode_example02/go",
		ChainCodeGoPath: os.Getenv("GOPATH"),

		PackArgs: func(params []string) [][]byte {
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
	c.initializeResourceClient()
	c.initializeChannelClient()
	c.initializeLedgerClient()
}

func (c *Client) initializeSDK() {
	sdk, err := fabsdk.New(config.FromFile(c.ConfigPath))
	if err != nil {
		utils.Fatalf("fabsdk.New err: %+v", err)
	}

	fmt.Println("Initialized fabric sdk")

	c.SDK = sdk
}

func (c *Client) initializeResourceClient() {
	clientProvider := c.SDK.Context(fabsdk.WithUser(c.OrgAdmin), fabsdk.WithOrg(c.OrgName))
	rc, err := resmgmt.New(clientProvider)
	if err != nil {
		utils.Fatalf("resmgmt.New err: %v", err)
	}

	fmt.Println("Initialized resource client")

	c.rc = rc
}

func (c *Client) initializeChannelClient() {
	channelProvider := c.SDK.ChannelContext(c.ChannelID, fabsdk.WithUser(c.OrgUser))

	cc, err := channel.New(channelProvider)
	if err != nil {
		utils.Fatalf("channel.New err: %v", err)
	}

	fmt.Println("Initialized channel client")

	c.cc = cc
}

func (c *Client) initializeLedgerClient() {
	channelProvider := c.SDK.ChannelContext(c.ChannelID, fabsdk.WithUser(c.OrgUser))
	lc, err := ledger.New(channelProvider)
	if err != nil {
		utils.Fatalf("ledger.New err: %v", err)
	}

	fmt.Println("Initialized ledger client")

	c.lc = lc
}

func (c *Client) Close() {
	c.SDK.Close()
}
