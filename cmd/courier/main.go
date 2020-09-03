package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/icodezjb/fabric-study/courier"
	"github.com/icodezjb/fabric-study/courier/client"
	"github.com/icodezjb/fabric-study/courier/utils"
	"github.com/icodezjb/fabric-study/log"

	"github.com/spf13/cobra"
)

func main() {
	var mainCmd = &cobra.Command{
		Use: "courier",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
			glogger.Verbosity(log.LvlDebug)
			log.Root().SetHandler(glogger)
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			h, err := courier.New(client.InitConfig())
			if err != nil {
				utils.Fatalf("[main] courier init err: %v", err)
			}

			h.Start()

			log.Info("[Main] courier service started")
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
			defer signal.Stop(interrupt)
			<-interrupt

			h.Stop()
			log.Info("[Main] courier service stopped")
		},
	}

	flags := mainCmd.PersistentFlags()
	client.InitConfigFile(flags)
	client.InitChannelID(flags)
	client.InitChaincodeID(flags)
	client.InitPeerURL(flags)
	client.InitUserName(flags)
	client.InitFilterEvents(flags)

	if err := mainCmd.Execute(); err != nil {
		fmt.Println(err)
	}

}
