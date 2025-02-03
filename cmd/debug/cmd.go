package debug

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {

	DebugCmd := &cobra.Command{
		Use:   "debug",
		Short: "Debugging tools",
	}

	DebugCmd.AddCommand(
		refactorCommand(),
		parseCommand(),
		agentCommand(),
		locateCommand(),
		repairCommand(),
	)
	return DebugCmd
}
