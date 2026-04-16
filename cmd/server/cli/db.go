package cli

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/FlameInTheDark/emerald/internal/config"
	"github.com/FlameInTheDark/emerald/internal/db"
	"github.com/urfave/cli/v3"
)

func RunMigrations(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("Migrations completed successfully")
	return nil
}

func DBVersion(ctx context.Context, cmd *cli.Command) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	dbPath := cfg.Database.Path
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return fmt.Errorf("database file does not exist at %s", dbPath)
	}

	database, err := db.New(cfg.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	var version int
	err = database.DB.QueryRow("SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No migrations have been run yet")
			return nil
		}
		return fmt.Errorf("failed to get version: %w", err)
	}

	fmt.Printf("Database version: %d\n", version)
	return nil
}
