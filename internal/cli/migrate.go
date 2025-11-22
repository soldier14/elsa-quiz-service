package cli

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"elsa-quiz-service/internal/config"
	pgmigrations "elsa-quiz-service/internal/infra/postgres/migrations"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"
)

// NewMigrateCmd applies database migrations.
func NewMigrateCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations(cmd.Context(), *configPath)
		},
	}
}

func runMigrations(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	return runMigrationsWithConfig(ctx, cfg)
}

func runMigrationsWithConfig(ctx context.Context, cfg config.Config) error {
	if cfg.Postgres.URL == "" {
		return fmt.Errorf("postgres url not configured")
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(cfg.Postgres.URL)))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	migrator := migrate.NewMigrator(db, pgmigrations.Migrations)

	if err := migrator.Init(ctx); err != nil {
		return err
	}

	if _, err := migrator.Migrate(ctx); err != nil {
		return err
	}
	log.Printf("migrations applied")
	return nil
}
