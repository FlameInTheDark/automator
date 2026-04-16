package cli

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/FlameInTheDark/emerald/internal/config"
	"github.com/FlameInTheDark/emerald/internal/db"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/urfave/cli/v3"
)

func DebugSQL(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dbPath := cfg.Database.Path
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist at %s. Run 'emerald db migrate' first", dbPath)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	fmt.Println("Emerald SQL Interactive Shell")
	fmt.Println("Type .help for commands, .exit to exit")
	fmt.Println()

	return runSQLShell(database.DB)
}

func runSQLShell(db *sql.DB) error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("emerald-sql> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if line == ".exit" || line == ".quit" {
			break
		}

		if line == ".help" {
			printSQLHelp()
			continue
		}

		if line == ".tables" {
			printTables(db)
			continue
		}

		if strings.HasPrefix(line, ".schema") {
			printSchema(db, strings.TrimPrefix(line, ".schema "))
			continue
		}

		if strings.HasPrefix(line, ".") {
			fmt.Printf("Unknown command: %s\n", line)
			continue
		}

		if err := executeQuery(db, line); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	fmt.Println("Goodbye!")
	return nil
}

func printSQLHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  .tables           - List all tables")
	fmt.Println("  .schema <table>   - Show schema for a table")
	fmt.Println("  .help             - Show this help")
	fmt.Println("  .exit, .quit      - Exit the shell")
	fmt.Println()
	fmt.Println("SQL statements are executed directly against the database.")
}

func printTables(db *sql.DB) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tableNames = append(tableNames, name)
	}

	if len(tableNames) == 0 {
		fmt.Println("No tables found")
		return
	}

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Tables"})
	for _, name := range tableNames {
		t.AppendRow(table.Row{name})
	}
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Render()
}

func printSchema(db *sql.DB, tableName string) {
	if tableName == "" {
		fmt.Println("Usage: .schema <table>")
		return
	}

	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	t := table.NewWriter()
	t.AppendHeader(table.Row{"Column", "Type", "Nullable", "PK"})
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, dfltVal int
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltVal, &pk); err != nil {
			continue
		}
		nullable := "YES"
		if notnull == 1 {
			nullable = "NO"
		}
		t.AppendRow(table.Row{name, colType, nullable, pk})
	}
	t.SetOutputMirror(os.Stdout)
	t.SetStyle(table.StyleDefault)
	t.Render()
}

func executeQuery(db *sql.DB, query string) error {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	if strings.HasPrefix(upperQuery, "SELECT") || strings.HasPrefix(upperQuery, "PRAGMA") {
		rows, err := db.Query(query)
		if err != nil {
			return err
		}
		defer func() {
			_ = rows.Close()
		}()

		columns, err := rows.Columns()
		if err != nil {
			return err
		}

		if len(columns) == 0 {
			fmt.Println("No results")
			return nil
		}

		t := table.NewWriter()
		headerRow := make(table.Row, len(columns))
		for i, col := range columns {
			headerRow[i] = col
		}
		t.AppendHeader(headerRow)

		for rows.Next() {
			values := make([]any, len(columns))
			valuePtrs := make([]any, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			if err := rows.Scan(valuePtrs...); err != nil {
				return err
			}

			row := make(table.Row, len(columns))
			for i, v := range values {
				if v == nil {
					row[i] = "NULL"
				} else {
					str := fmt.Sprintf("%v", v)
					if len(str) > 100 {
						str = str[:97] + "..."
					}
					row[i] = str
				}
			}
			t.AppendRow(row)
		}

		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleDefault)
		t.Render()
		fmt.Printf("\n(%d rows)\n", t.Length())
	} else {
		result, err := db.Exec(query)
		if err != nil {
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		fmt.Printf("OK, %d rows affected\n", rowsAffected)
	}

	return nil
}
