package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/icodezjb/fabric-study/courier"
	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/log"

	"github.com/spf13/cobra"
)

func main() {
	var mainCmd = &cobra.Command{
		Use: "courier",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			h := courier.New(client.InitConfig())

			h.Start()

			log.Info("courier service start")
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(interrupt)
			<-interrupt

			h.Stop()
			log.Info("courier service stop")
		},
	}

	flags := mainCmd.PersistentFlags()
	client.InitConfigFile(flags)
	client.InitChannelID(flags)
	client.InitChaincodeID(flags)
	client.InitPeerURL(flags)
	client.InitUserName(flags)

	if err := mainCmd.Execute(); err != nil {
		fmt.Println(err)
	}

}
