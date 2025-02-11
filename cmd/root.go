package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/nakamasato/aicoder/cmd/apply"
	cmdconfig "github.com/nakamasato/aicoder/cmd/config"
	"github.com/nakamasato/aicoder/cmd/db"
	"github.com/nakamasato/aicoder/cmd/debug"
	"github.com/nakamasato/aicoder/cmd/gh"
	"github.com/nakamasato/aicoder/cmd/load"
	"github.com/nakamasato/aicoder/cmd/plan"
	"github.com/nakamasato/aicoder/cmd/review"
	"github.com/nakamasato/aicoder/cmd/search"
	"github.com/nakamasato/aicoder/cmd/summarize"
	"github.com/nakamasato/aicoder/config"
)

var configFile string

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aicoder",
		Short: "Aicoder is a AI-powered CLI tool that helps you to code quickly.",
		Long:  `Aicoder is a AI-powered CLI tool that helps you to code quickly.`,
	}

	cobra.OnInitialize(func() {
		log.Println(cmd.CalledAs())
		if cmd.CalledAs() != "config" {
			initConfig()
		}
	})

	cmd.PersistentFlags().StringVar(&configFile, "config", ".aicoder.yaml", "config file (default is .aicoder.yaml)")

	// Add commands
	cmd.AddCommand(
		apply.Command(),
		cmdconfig.Command(),
		db.Command(),
		debug.Command(),
		gh.NewGhCmd(),
		load.Command(),
		plan.Command(),
		review.Command(),
		search.Command(),
		summarize.Command(),
	)

	return cmd
}

// Execute executes the root command.
func Execute() error {
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
	return nil
}

func initConfig() {
	config.InitConfig(configFile)
}
