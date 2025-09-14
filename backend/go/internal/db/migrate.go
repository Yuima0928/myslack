package db

import (
	"database/sql"
	"fmt"

	"slackgo/internal/db/migrations"

	"github.com/pressly/goose/v3"
)

// RunMigrations runs goose.Up using embedded migrations.
func RunMigrations(sqlDB *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	// 埋め込みFSを使う
	goose.SetBaseFS(migrations.FS)
	// カレント相対で探す（embed経由なのでディレクトリは "." でOK）
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
