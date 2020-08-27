package main

import (
	"time"

	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/spf13/cobra"
)

var mainCmd = &cobra.Command{
	Use: "courier",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func main() {

	flags := mainCmd.PersistentFlags()
	client.InitConfigFile(flags)
	client.InitChannelID(flags)
	client.InitChaincodeID(flags)
	client.InitPeerURL(flags)
	client.InitUserName(flags)

	c := client.New(client.InitConfig())
	defer c.Close()

	blkSync := client.NewBlockSync(c)
	defer blkSync.Stop()

	time.Sleep(60 * time.Minute)
}
