/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package db

import (
	"github.com/spf13/cobra"
)

var dbConnString string

func Command() *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "db",
		Long:  `db`,
	}
	dbCmd.AddCommand(
		migrateCommand(),
	)
	return dbCmd
}
