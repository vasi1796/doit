package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close database: %v\n", err)
		}
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set dialect: %v\n", err)
		os.Exit(1)
	}

	if err := goose.RunContext(context.Background(), command, db, migrationsDir, os.Args[2:]...); err != nil {
		fmt.Fprintf(os.Stderr, "migration %s failed: %v\n", command, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: migrate <command>")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  up        Migrate to the latest version")
	fmt.Fprintln(os.Stderr, "  down      Roll back one migration")
	fmt.Fprintln(os.Stderr, "  status    Show migration status")
	fmt.Fprintln(os.Stderr, "  reset     Roll back all migrations")
	fmt.Fprintln(os.Stderr, "  redo      Roll back and re-apply the latest migration")
}
