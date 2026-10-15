package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	_ "modernc.org/sqlite" // Registers the "sqlite" driver

	"github.com/spf13/cobra"
)

var (
	ctx          context.Context
	dbFile       string
	db           *sql.DB
	showDateTime bool
	showID       bool
	limitFlag    int
)

var rootCmd = &cobra.Command{
	Use:   "gh",
	Short: "GhistInTime - global history tracker",
	RunE:  dumpAll,
	//PreRunE: func(cmd *cobra.Command, args []string) error {
	//	if formatFlag != "" {
	//		if !slices.Contains(allowedFormats, formatFlag) {
	//			return fmt.Errorf("invalid format %q: must be one of %v", formatFlag, allowedFormats)
	//		}
	//	}
	//	return nil
	//},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {

		if dbFile == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				slog.Error("failed to get user home directory - please set db file")
				return err
			}
			dbFile = fmt.Sprintf("%s/.ghist2.db", home)
		}
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

//var dumpCmd = &cobra.Command{
//	Use:   "dump",
//	Short: "Dump all",
//	RunE:  dumpAll,
//	PreRunE: func(cmd *cobra.Command, args []string) error {
//		if formatFlag != "" {
//			if !slices.Contains(allowedFormats, formatFlag) {
//				return fmt.Errorf("invalid format %q: must be one of %v", formatFlag, allowedFormats)
//			}
//		}
//		return nil
//	},
//}

func addLine(cmd *cobra.Command, args []string) error {
	deleteOld := "DELETE FROM ghist WHERE command = ?"
	line := strings.TrimSpace(args[0])
	if _, err := db.ExecContext(ctx, deleteOld, line); err != nil {
		slog.Error(fmt.Sprintf("Failed to delete from table: %v", err))
		return err
	}
	insertNew := "INSERT INTO ghist(command, inserted) VALUES(?, strftime('%s','now'))"
	if _, err := db.ExecContext(ctx, insertNew, line); err != nil {
		slog.Error(fmt.Sprintf("Failed to add new row to table: %v", err))
		return err
	}
	return nil
}

func dumpAll(cmd *cobra.Command, args []string) error {
	var limStr string
	//if formatFlag == "long" {
	//	colStr = "id, datetime(inserted, 'unixepoch', 'localtime') as dt, command"
	//} else if formatFlag == "dt" {
	//	colStr = "datetime(inserted, 'unixepoch', 'localtime') as dt, id, command"
	//} else {
	//	colStr = "id, command"
	//}
	//cols := []string{"command"}
	//if showID {
	//	cols = append([]string{"id"}, cols...)
	//}
	//if showDateTime {
	//	cols = append([]string{ "datetime(inserted, 'unixepoch', 'localtime') as dt"}, cols...)
	//}
	//colStr = strings.Join(cols, ",")

	//..colStr = "datetime(inserted, 'unixepoch', 'localtime') as dt, id, command"
	if limitFlag >= 0 {
		limStr = fmt.Sprintf("LIMIT %d", limitFlag)
	}
	query := fmt.Sprintf(`
	SELECT %s
	FROM ghist
	ORDER BY id
	%s
	`, ColStr(), limStr)

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
	//if formatFlag == "" {
	//	fmt.Fprintln(w, strings.Join(cols, "\t"))
	//}

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

func ColStr() string {
	cols := []string{"command"}
	if showID {
		cols = append([]string{"id"}, cols...)
	}
	if showDateTime {
		cols = append([]string{"datetime(inserted, 'unixepoch', 'localtime') as dt"}, cols...)
	}
	return strings.Join(cols, ",")
}

func init() {
	rootCmd.AddCommand(addCmd)
	//rootCmd.AddCommand(dumpCmd)

	rootCmd.PersistentFlags().StringVarP(&dbFile, "db", "", "", "Database")

	//rootCmd.Flags().StringVarP(&formatFlag, "fmt", "f", "", "Format")
	rootCmd.Flags().IntVarP(&limitFlag, "limit", "l", -1, "Format")
	rootCmd.Flags().BoolVarP(&showDateTime, "dt", "d", false, "Format")
	rootCmd.Flags().BoolVarP(&showID, "id", "i", false, "Format")

	//_ = rootCmd.MarkPersistentFlagRequired("db")
}

func main() {
	for i, arg := range os.Args {
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			if _, err := strconv.Atoi(arg[1:]); err == nil {
				os.Args[i] = "--limit=" + arg[1:]
				continue
			}
		}
	}

	ctx = context.Background()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
