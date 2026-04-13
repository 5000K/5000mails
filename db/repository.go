package db

import (
	"log/slog"

	"gorm.io/gorm"
)

type MailingListRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func NewMailingListRepository(db *gorm.DB, logger *slog.Logger) *MailingListRepository {
	return &MailingListRepository{
		db:     db,
		logger: logger,
	}
}
