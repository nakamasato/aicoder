package db

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/migrate"
	"github.com/spf13/cobra"
)

func migrateCommand() *cobra.Command {
	dbCmd := &cobra.Command{
		Use:   "migrate",
		Short: "ent migration",
		Long:  `ent migration`,
		Run:   dbMigrate,
	}
	dbCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")
	return dbCmd
}

func dbMigrate(cmd *cobra.Command, args []string) {
	fmt.Println("db migrate")
	ctx := cmd.Context()
	entClient, err := ent.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()
	if err := entClient.Schema.Create(ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(true),
	); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	fmt.Println("db migrate done")
}
