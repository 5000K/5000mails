package db

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) AddUser(ctx context.Context, mailingListID uint, name, email, unsubscribeToken string) (*domain.User, error) {
	user := &User{
		Name:             name,
		Email:            email,
		MailingListID:    mailingListID,
		UnsubscribeToken: unsubscribeToken,
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

func (r *MailingListRepository) GetUserByUnsubscribeToken(ctx context.Context, token string) (*domain.User, error) {
	var user User

	result := r.db.WithContext(ctx).Where("unsubscribe_token = ?", token).First(&user)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get user by unsubscribe token",
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get user by unsubscribe token: %w", result.Error)
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

func (r *MailingListRepository) GetUsers(ctx context.Context, mailingListID uint) ([]domain.User, error) {
	var users []User

	result := r.db.WithContext(ctx).Where("mailing_list_id = ?", mailingListID).Find(&users)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get users",
			slog.Uint64("mailing_list_id", uint64(mailingListID)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get users: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "fetched users",
		slog.Uint64("mailing_list_id", uint64(mailingListID)),
		slog.Int("count", len(users)),
	)
	return ToDomainUsers(users), nil
}

func (r *MailingListRepository) CreateConfirmation(ctx context.Context, userID uint, token string) (*domain.Confirmation, error) {
	c := &Confirmation{UserID: userID, Token: token}

	result := r.db.WithContext(ctx).Create(c)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to create confirmation",
			slog.Uint64("user_id", uint64(userID)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("create confirmation: %w", result.Error)
	}

	r.logger.InfoContext(ctx, "created confirmation",
		slog.Uint64("user_id", uint64(userID)),
		slog.Uint64("confirmation_id", uint64(c.ID)),
	)
	return ToDomainConfirmation(c), nil
}

func (r *MailingListRepository) GetConfirmationByToken(ctx context.Context, token string) (*domain.Confirmation, error) {
	var c Confirmation

	result := r.db.WithContext(ctx).Where("token = ?", token).First(&c)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get confirmation by token",
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get confirmation by token: %w", result.Error)
	}

	return ToDomainConfirmation(&c), nil
}

func (r *MailingListRepository) DeleteConfirmation(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&Confirmation{}, id)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to delete confirmation",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("delete confirmation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("delete confirmation: confirmation %d not found", id)
	}
	r.logger.InfoContext(ctx, "deleted confirmation",
		slog.Uint64("id", uint64(id)),
	)
	return nil
}
