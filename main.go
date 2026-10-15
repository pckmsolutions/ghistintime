package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"text/tabwriter"

	_ "modernc.org/sqlite" // Registers the "sqlite" driver

	"github.com/spf13/cobra"
)

var (
	ctx            context.Context
	dbFile         string
	db             *sql.DB
	formatFlag     string
	allowedFormats = []string{"hist", "histtime"}
)

var rootCmd = &cobra.Command{
	Use:   "gh",
	Short: "GhistInTime - global history tracker",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		db, err = sql.Open("sqlite", dbFile)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to open database: %v", err))
			return err
		}

		createTableSQL := `
        CREATE TABLE IF NOT EXISTS ghist(
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            command TEXT NOT NULL UNIQUE,
            inserted INTEGER NOT NULL
        )
				`
		if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
			slog.Error(fmt.Sprintf("Failed to create table: %v", err))
			return err
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if db != nil {
			db.Close()
		}

		return nil
	},
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add line",
	Args:  cobra.MinimumNArgs(1),
	RunE:  addLine}

var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump all",
	RunE:  dumpAll,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if formatFlag != "" {
			if !slices.Contains(allowedFormats, formatFlag) {
				return fmt.Errorf("invalid format %q: must be one of %v", formatFlag, allowedFormats)
			}
		}
		return nil
	},
}

func addLine(cmd *cobra.Command, args []string) error {
	deleteOld := "DELETE FROM ghist WHERE command = ?"
	if _, err := db.ExecContext(ctx, deleteOld, args[0]); err != nil {
		slog.Error(fmt.Sprintf("Failed to delete from table: %v", err))
		return err
	}
	insertNew := "INSERT INTO ghist(command, inserted) VALUES(?, strftime('%s','now'))"
	if _, err := db.ExecContext(ctx, insertNew, args[0]); err != nil {
		slog.Error(fmt.Sprintf("Failed to add new row to table: %v", err))
		return err
	}
	return nil
}

func dumpAll(cmd *cobra.Command, args []string) error {
	var colStr string
	if formatFlag == "hist" {
		colStr = "id, command"
	} else if formatFlag == "histtime" {
		colStr = "datetime(inserted, 'unixepoch', 'localtime') as dt, id, command"
	} else {
		colStr = "command, id, datetime(inserted, 'unixepoch', 'localtime') as dt"
	}
	query := fmt.Sprintf(`
	SELECT %s
	FROM ghist
	ORDER BY id
	`, colStr)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	// Set up tabwriter directly writing to stdout: minwidth, tabwidth, padding, padchar, flags
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print headers tab-separated
	if formatFlag == "" {
		fmt.Fprintln(w, strings.Join(cols, "\t"))
	}

	values := make([]interface{}, len(cols))
	valuePtrs := make([]interface{}, len(cols))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		rowStr := make([]string, len(cols))
		for i, val := range values {
			if val == nil {
				rowStr[i] = "NULL"
			} else if b, ok := val.([]byte); ok {
				rowStr[i] = string(b)
			} else {
				rowStr[i] = fmt.Sprintf("%v", val)
			}
		}

		fmt.Fprintln(w, strings.Join(rowStr, "\t"))
	}

	return rows.Err()
}

func init() {
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(dumpCmd)

	rootCmd.PersistentFlags().StringVarP(&dbFile, "db", "", "", "Database")

	dumpCmd.Flags().StringVarP(&formatFlag, "fmt", "f", "", "Format")

	_ = rootCmd.MarkPersistentFlagRequired("db")
}

func main() {
	ctx = context.Background()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
