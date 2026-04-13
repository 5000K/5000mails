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

	r.logger.InfoContext(ctx, "created mailing list",
		slog.String("name", name),
		slog.Uint64("id", uint64(list.ID)),
	)
	return ToDomainList(list), nil
}

func (r *MailingListRepository) GetList(ctx context.Context, id uint) (*domain.MailingList, error) {
	var list MailingList

	result := r.db.WithContext(ctx).First(&list, id)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get mailing list",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get mailing list: %w", result.Error)
	}

	return ToDomainList(&list), nil
}

func (r *MailingListRepository) GetListByName(ctx context.Context, name string) (*domain.MailingList, error) {
	var list MailingList

	result := r.db.WithContext(ctx).Where("name = ?", name).First(&list)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get mailing list by name",
			slog.String("name", name),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get mailing list by name: %w", result.Error)
	}

	return ToDomainList(&list), nil
}

func (r *MailingListRepository) UpdateList(ctx context.Context, id uint, name string) (*domain.MailingList, error) {
	var list MailingList
	result := r.db.WithContext(ctx).First(&list, id)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to find mailing list for update",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("update mailing list: %w", result.Error)
	}

	list.Name = name
	result = r.db.WithContext(ctx).Save(&list)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to update mailing list",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("update mailing list: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "updated mailing list",
		slog.Uint64("id", uint64(id)),
		slog.String("name", name),
	)
	return ToDomainList(&list), nil
}

func (r *MailingListRepository) DeleteList(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&MailingList{}, id)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to delete mailing list",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("delete mailing list: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete mailing list: list %d not found", id)
	}
	r.logger.InfoContext(ctx, "deleted mailing list",
		slog.Uint64("id", uint64(id)),
	)
	return nil
}
