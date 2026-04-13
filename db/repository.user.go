package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) AddUser(ctx context.Context, mailingListID uint, name, email string) (*domain.User, error) {
	user := &User{
		Name:          name,
		Email:         email,
		MailingListID: mailingListID,
	}

	result := r.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to add user to mailing list",
			slog.Uint64("mailing_list_id", uint64(mailingListID)),
			slog.String("email", email),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("add user: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "added user to mailing list",
		slog.Uint64("mailing_list_id", uint64(mailingListID)),
		slog.Uint64("user_id", uint64(user.ID)),
		slog.String("email", email),
	)
	return ToDomainUser(user), nil
}

func (r *MailingListRepository) ConfirmUser(ctx context.Context, userID uint) error {
	now := time.Now()

	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ? AND confirmed_at IS NULL", userID).
		Update("confirmed_at", &now)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to confirm user",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("confirm user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		r.logger.WarnContext(ctx, "confirm user had no effect: already confirmed or not found",
			slog.Uint64("user_id", uint64(userID)),
		)
		return fmt.Errorf("confirm user: user %d not found or already confirmed", userID)
	}

	r.logger.InfoContext(ctx, "confirmed user subscription",
		slog.Uint64("user_id", uint64(userID)),
		slog.Time("confirmed_at", now),
	)
	return nil
}

func (r *MailingListRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user User

	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get user by email",
			slog.String("email", email),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get user by email: %w", result.Error)
	}

	return ToDomainUser(&user), nil
}

func (r *MailingListRepository) GetConfirmedUsers(ctx context.Context, mailingListID uint) ([]domain.User, error) {
	var users []User

	result := r.db.WithContext(ctx).
		Where("mailing_list_id = ? AND confirmed_at IS NOT NULL", mailingListID).
		Find(&users)

	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get confirmed users",
			slog.Uint64("mailing_list_id", uint64(mailingListID)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get confirmed users: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "fetched confirmed users",
		slog.Uint64("mailing_list_id", uint64(mailingListID)),
		slog.Int("count", len(users)),
	)
	return ToDomainUsers(users), nil
}

func (r *MailingListRepository) RemoveUser(ctx context.Context, userID uint) error {
	result := r.db.WithContext(ctx).Delete(&User{}, userID)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to remove user",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("remove user: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("remove user: user %d not found", userID)
	}

	r.logger.InfoContext(ctx, "removed user from mailing list",
		slog.Uint64("user_id", uint64(userID)),
	)
	return nil
}
