package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) CreateList(ctx context.Context, name string) (*domain.MailingList, error) {
	list := &MailingList{Name: name}

	result := r.db.WithContext(ctx).Create(list)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to create mailing list",
			slog.String("name", name),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("create mailing list: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "created mailing list", slog.String("name", name))
	return ToDomainList(list), nil
}

func (r *MailingListRepository) GetListByName(ctx context.Context, name string) (*domain.MailingList, error) {
	var list MailingList

	result := r.db.WithContext(ctx).First(&list, "name = ?", name)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get mailing list",
			slog.String("name", name),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get mailing list: %w", result.Error)
	}

	return ToDomainList(&list), nil
}

func (r *MailingListRepository) RenameList(ctx context.Context, name, newName string) (*domain.MailingList, error) {
	result := r.db.WithContext(ctx).
		Model(&MailingList{}).
		Where("name = ?", name).
		Update("name", newName)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to rename mailing list",
			slog.String("name", name),
			slog.String("new_name", newName),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("rename mailing list: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("rename mailing list: list %q not found", name)
	}

	r.logger.InfoContext(ctx, "renamed mailing list",
		slog.String("name", name),
		slog.String("new_name", newName),
	)
	return &domain.MailingList{Name: newName}, nil
}

func (r *MailingListRepository) DeleteList(ctx context.Context, name string) error {
	result := r.db.WithContext(ctx).Delete(&MailingList{Name: name})
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to delete mailing list",
			slog.String("name", name),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("delete mailing list: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete mailing list: list %q not found", name)
	}
	r.logger.InfoContext(ctx, "deleted mailing list", slog.String("name", name))
	return nil
}
