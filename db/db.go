package db

import (
	efs "app"
	"app/config"
	"database/sql"
	"log/slog"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/pressly/goose/v3"
)

var Bun *bun.DB

func InitPostgresDatabase() {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(config.Config.DatabaseURL)))
	Bun = bun.NewDB(sqldb, pgdialect.New())

	if os.Getenv("ENV") != "dev" {
		Migrate()
	}
}

func Migrate() {
	goose.SetBaseFS(efs.MigrationsFS)

	err := goose.Up(Bun.DB, "migrations")
	if err != nil {
		panic(err)
	}

	slog.Info("database migrated")
}
