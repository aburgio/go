package cmd

import (
	"fmt"
	stdLog "log"
	"os"

	"github.com/spf13/cobra"
	frontier "github.com/digitalbits/go/services/frontier/internal"
)

var (
	config, flags = frontier.Flags()

	rootCmd = &cobra.Command{
		Use:   "frontier",
		Short: "client-facing api server for the digitalbits network",
		Long:  "client-facing api server for the digitalbits network. It acts as the interface between DigitalBits Core and applications that want to access the DigitalBits network. It allows you to submit transactions to the network, check the status of accounts, subscribe to event streams and more.",
		Run: func(cmd *cobra.Command, args []string) {
			frontier.NewAppFromFlags(config, flags).Serve()
		},
	}
)

func init() {
	err := flags.Init(rootCmd)
	if err != nil {
		stdLog.Fatal(err.Error())
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}