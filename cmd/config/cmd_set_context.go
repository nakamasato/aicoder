package config

import (
	"log"

	"github.com/spf13/cobra"
)

func setCommand() *cobra.Command {
	cmdSetContext := &cobra.Command{
		Use:   "setcontext [context]",
		Short: "Set current context",
		Run:   runSetCurrentContext,
		Args:  cobra.MinimumNArgs(1),
	}

	return cmdSetContext
}

func runSetCurrentContext(cmd *cobra.Command, args []string) {
	currentContext := args[0]
	if currentContext == "" {
		log.Fatalln("current context is required")
	}

	cmd.Printf("set current context %s\n", currentContext)

	// update config file if exists

	cmd.Println("set current context not implemented yet.")
}
