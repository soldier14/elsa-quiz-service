package migrations

import (
	"context"
	_ "embed"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

//go:embed 0001_create_quizzes.sql
var createQuizzesSQL string

var Migrations = migrate.NewMigrations()

func init() {
	Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(createQuizzesSQL)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`DROP TABLE IF EXISTS quizzes`)
			return err
		},
	)
}
