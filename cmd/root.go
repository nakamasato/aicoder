/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/nakamasato/aicoder/cmd/db"
	"github.com/nakamasato/aicoder/cmd/loader"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "aicoder",
	Short: "Aicoder helps you to code quickly",
	Long:  `Aicoder is a CLI tool that helps you to code quickly.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	RootCmd.AddCommand(
		loader.Command(),
		db.Command(),
	)
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
