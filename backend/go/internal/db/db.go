package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"slackgo/internal/model"
)

func Open(dsn string) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := model.EnableExtensions(gdb); err != nil {
		return nil, err
	}
	return gdb, nil
}
