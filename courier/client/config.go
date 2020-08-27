package client

import (
	"strings"

	"github.com/icodezjb/fabric-study/courier/utils"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/core"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/spf13/pflag"
)

const (
	UserFlag        = "user"
	userDescription = "The user"
	defaultUser     = "User1"

	ChannelIDFlag        = "cid"
	channelIDDescription = "The channel ID"
	defaultChannelID     = ""

	ChaincodeIDFlag        = "ccid"
	chaincodeIDDescription = "The Chaincode ID"
	defaultChaincodeID     = ""

	PeerURLFlag        = "peer"
	peerURLDescription = "A comma-separated list of peer targets, e.g. 'grpcs://localhost:7051,grpcs://localhost:8051'"
	defaultPeerURL     = ""

	ConfigFileFlag        = "config"
	configFileDescription = "The path of the config.yaml file"
	defaultConfigFile     = ""
)

type options struct {
	configFile string
	peerUrl    string

	User        string
	ChannelID   string
	ChainCodeID string
}

type Config struct {
	core.ConfigProvider
	channel.RequestOption
}

var opts options

// InitUserName initializes the user name from the provided arguments
func InitUserName(flags *pflag.FlagSet) {
	flags.StringVar(&opts.User, UserFlag, defaultUser, userDescription)
}

// InitChannelID initializes the channel ID from the provided arguments
func InitChannelID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.ChannelID, ChannelIDFlag, defaultChannelID, channelIDDescription)
}

// InitChaincodeID initializes the chaincode ID from the provided arguments
func InitChaincodeID(flags *pflag.FlagSet) {
	flags.StringVar(&opts.ChainCodeID, ChaincodeIDFlag, defaultChaincodeID, chaincodeIDDescription)
}

// InitPeerURL initializes the peer URL from the provided arguments
func InitPeerURL(flags *pflag.FlagSet) {
	flags.StringVar(&opts.peerUrl, PeerURLFlag, defaultPeerURL, peerURLDescription)
}

// InitConfigFile initializes the config file path from the provided arguments
func InitConfigFile(flags *pflag.FlagSet) {
	flags.StringVar(&opts.configFile, ConfigFileFlag, defaultConfigFile, configFileDescription)
}

func peerURLs() []string {
	if opts.peerUrl == "" {
		utils.Fatalf("peer not set")
	}

	var urls []string
	if len(strings.TrimSpace(opts.peerUrl)) > 0 {
		peerUrls := strings.Split(opts.peerUrl, ",")
		for _, url := range peerUrls {
			urls = append(urls, url)
		}
	}

	return urls
}

// InitConfig initializes the configuration
func InitConfig() *Config {
	cnfg := config.FromFile(opts.configFile)

	cfg := &Config{
		ConfigProvider: cnfg,
		RequestOption:  channel.WithTargetEndpoints(peerURLs()...),
	}

	return cfg
}

// InitUserName initializes the user name from the provided arguments
func (c *Config) UserName() string {
	if opts.User == "" {
		utils.Fatalf("user not set")
	}

	return opts.User
}

// ChannelID returns the channel ID
func (c *Config) ChannelID() string {
	if opts.User == "" {
		utils.Fatalf("cid not set")
	}

	return opts.ChannelID
}

// ChaincodeID returns the chaicode ID
func (c *Config) ChainCodeID() string {
	if opts.ChannelID == "" {
		utils.Fatalf("ccid not set")
	}

	return opts.ChainCodeID
}

// PeerURLs returns a list of peer URLs
func (c *Config) PeerURLs() []string {
	return peerURLs()
}
