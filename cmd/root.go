/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/nakamasato/aicoder/cmd/apply"
	"github.com/nakamasato/aicoder/cmd/db"
	"github.com/nakamasato/aicoder/cmd/load"
	"github.com/nakamasato/aicoder/cmd/plan"
	"github.com/nakamasato/aicoder/cmd/search"
	"github.com/nakamasato/aicoder/config"
	"github.com/spf13/cobra"
)

var (
	cfgFile string

	RootCmd = &cobra.Command{
		Use:   "aicoder",
		Short: "Aicoder is a AI-powered CLI tool that helps you to code quickly.",
		Long:  `Aicoder is a AI-powered CLI tool that helps you to code quickly.`,
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.AddCommand(
		load.Command(),
		db.Command(),
		search.Command(),
		plan.Command(),
		apply.Command(),
	)
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".aicoder.yaml", "config file (default is .aicoder.yaml)")
}

func initConfig() {
	config.ReadConfig(cfgFile)
}
