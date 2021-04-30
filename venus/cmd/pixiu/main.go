package main

import (
	"os"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vjoke/falcon/venus/pkg/bootstrap"
	"github.com/vjoke/falcon/venus/pkg/cmd"
	"github.com/vjoke/falcon/pkg/log"
)

var (
	botArgs *bootstrap.PixiuArgs

	loggingOptions = log.DefaultOptions()

	rootCmd = &cobra.Command{
		Use:          "venus-pixiu",
		Short:        "Falcon Venus.",
		Long:         "A hunter for profit from crypto-trading.",
		SilenceUsage: true,
	}

	pixiuCmd = &cobra.Command{
		Use:               "bot",
		Short:             "Start pixiu service.",
		Args:              cobra.ExactArgs(0),
		PersistentPreRunE: configureLogging,
		PreRunE: func(c *cobra.Command, args []string) error {
			// TODO: validate configure
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			cmd.PrintFlags(c.Flags())

			// Create the stop channel for all of the servers.
			stop := make(chan struct{})

			// Create the server for the discovery service.
			pixiu, err := bootstrap.NewBot(botArgs)
			if err != nil {
				return fmt.Errorf("failed to create pixiu service: %v", err)
			}

			// Start the server
			if err := pixiu.Start(stop); err != nil {
				return fmt.Errorf("failed to start pixiu service: %v", err)
			}

			cmd.WaitSignal(stop)
			// Wait until we shut down. In theory this could block forever; in practice we will get
			// forcibly shut down after 30s in Kubernetes.
			pixiu.WaitUntilCompletion()
			return nil
		},
	}
)

func configureLogging(_ *cobra.Command, _ []string) error {
	if err := log.Configure(loggingOptions); err != nil {
		return err
	}
	return nil
}

func init() {
	botArgs = bootstrap.NewPixiuArgs(func(p *bootstrap.PixiuArgs) {
		// TODO:
	})

	// Process commandline args.
	pixiuCmd.PersistentFlags().StringVar(&botArgs.ConfigFile, "config", "./config/binance/normal-policy.toml",
		"Config file name for trading. If not specified, a default config file will be used.")

	// Attach the pixiu logging options to the command.
	loggingOptions.AttachCobraFlags(rootCmd)

	cmd.AddFlags(rootCmd)

	rootCmd.AddCommand(pixiuCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(-1)
	}
}