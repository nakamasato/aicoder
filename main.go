package main

import (
	"os"

	"github.com/nakamasato/aicoder/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
