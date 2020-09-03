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
	configFileDescription = "The path of the config.yaml file needed by fabric-sdk-go"
	defaultConfigFile     = ""

	filterEventFlag        = "events"
	filterEventDescription = "A comma-separated list of the specified events which are in the fabric blocks, e.g. 'precommit, commit'"
	defaultFilterEvent     = "precommit,commit"

	HTTPEndpointFlag            = "endpoint"
	HTTPEndpointFlagDescription = "The courier http server listening, e.g. 'localhost:8080'"
	defaultHTTPEndpointFlag     = "localhost:8080"

	DataDirFlag            = "datadir"
	DataDirFlagDescription = "The courier data directory"
	defaultDataDirFlag     = "./courier_data"
)

type options struct {
	configFile string
	peerUrl    string
	events     string

	User        string
	ChannelID   string
	ChainCodeID string

	HTTPEndpoint string
	DataDir      string
}

type Config struct {
	// fabric client config
	core.ConfigProvider
	channel.RequestOption
	FilterEvents []string
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

// InitFilterEvents initializes the filter events from the provided arguments
func InitFilterEvents(flags *pflag.FlagSet) {
	flags.StringVar(&opts.events, filterEventFlag, defaultFilterEvent, filterEventDescription)
}

// HTTPEndpoint initializes the courier http server listening from the provided arguments
func InitHTTPEndpoint(flags *pflag.FlagSet) {
	flags.StringVar(&opts.HTTPEndpoint, HTTPEndpointFlag, defaultHTTPEndpointFlag, HTTPEndpointFlagDescription)
}

// InitDataDir initializes the courier data directory from the provided arguments
func InitDataDir(flags *pflag.FlagSet) {
	flags.StringVar(&opts.DataDir, DataDirFlag, defaultDataDirFlag, DataDirFlagDescription)
}

func peerURLs() []string {
	if opts.peerUrl == "" {
		utils.Fatalf("[Config] peer not set")
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

func filterEvents() []string {
	if opts.events == "" {
		utils.Fatalf("[Config] filter events not set")
	}

	var filterEvents []string
	if len(strings.TrimSpace(opts.events)) > 0 {
		events := strings.Split(opts.events, ",")
		for _, ev := range events {
			filterEvents = append(filterEvents, ev)
		}
	}

	return filterEvents
}

// InitConfig initializes the configuration
func InitConfig() *Config {
	cnfg := config.FromFile(opts.configFile)

	cfg := &Config{
		ConfigProvider: cnfg,
		RequestOption:  channel.WithTargetEndpoints(peerURLs()...),
		FilterEvents:   filterEvents(),
	}

	return cfg
}

// InitUserName initializes the user name from the provided arguments
func (c *Config) UserName() string {
	if opts.User == "" {
		utils.Fatalf("[Config] user not set")
	}

	return opts.User
}

// ChannelID returns the channel ID
func (c *Config) ChannelID() string {
	if opts.User == "" {
		utils.Fatalf("[Config] cid not set")
	}

	return opts.ChannelID
}

// ChainCodeID returns the chaicode ID
func (c *Config) ChainCodeID() string {
	if opts.ChannelID == "" {
		utils.Fatalf("[Config] ccid not set")
	}

	return opts.ChainCodeID
}

// PeerURLs returns a list of peer URLs
func (c *Config) PeerURLs() []string {
	return peerURLs()
}

// ServerURL returns the courier http server url
func (c *Config) HTTPEndpoint() string {
	return opts.HTTPEndpoint
}

// DataDir returns the courier data directory
func (c *Config) DataDir() string {
	return opts.DataDir
}
