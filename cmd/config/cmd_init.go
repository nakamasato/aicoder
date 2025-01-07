package config

import (
	"log"
	"os"

	"github.com/nakamasato/aicoder/config"
	"github.com/spf13/cobra"
)


func initCommand() *cobra.Command {
	cmdInit := &cobra.Command{
		Use:   "init",
		Short: "Create a default .aicoder.yaml configuration file",
		Run:   runInit,
	}
	return cmdInit
}

func runInit(cmd *cobra.Command, args []string) {
	if _, err := os.Stat(outputFile); err == nil {
		cmd.Println(".aicoder.yaml already exists")
		return
	}

	file, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	if err := config.CreateDefaultConfigFile(file); err != nil {
		log.Fatalf("Failed to create default configuration file: %v", err)
	}

	// Print success message
	cmd.Println("Default configuration file created at .aicoder.yaml")
}
