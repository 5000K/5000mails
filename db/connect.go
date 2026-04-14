package db

import (
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Connect(dbType string, dsn string) (*gorm.DB, error) {
	switch dbType {
	case "mysql":
		slog.Debug("connecting to mysql")
		return gorm.Open(mysql.Open(dsn), &gorm.Config{})

	case "postgres":
		slog.Debug("connecting to postgres")
		return gorm.Open(postgres.Open(dsn), &gorm.Config{})

	case "sqlite":
		slog.Debug("opening sqlite")
		return gorm.Open(sqlite.Open(dsn), &gorm.Config{})

	default:
		slog.Error("Unknown database type", "db-type", dbType)
		return nil, errors.New("unknown database type: " + dbType)
	}
}

func AutoMigrate(database *gorm.DB) error {
	if err := database.AutoMigrate(&MailingList{}, &User{}, &Confirmation{}, &SentNewsletter{}); err != nil {
		return fmt.Errorf("auto-migrating database: %w", err)
	}
	return nil
}
