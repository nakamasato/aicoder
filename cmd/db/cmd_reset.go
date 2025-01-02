package db

import (
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/nakamasato/aicoder/ent"
	"github.com/spf13/cobra"
)

func resetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "ent migration",
		Long:  `ent migration`,
		Run:   reset,
	}
	cmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")
	return cmd
}

func reset(cmd *cobra.Command, args []string) {
	fmt.Println("db reset")
	ctx := cmd.Context()
	entClient, err := ent.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()

	entClient.Document.Delete().ExecX(ctx)

	if err := entClient.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
	entClient.Document.Delete().ExecX(ctx)
	fmt.Println("db reset done")
}
