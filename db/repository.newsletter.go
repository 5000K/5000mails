package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) CreateSentNewsletter(ctx context.Context, subject, senderName, rawMarkdown string, recipientIDs []uint, listNames []string) (*domain.SentNewsletter, error) {
	recipients := make([]User, len(recipientIDs))
	for i, id := range recipientIDs {
		recipients[i] = User{}
		recipients[i].ID = id
	}

	mailingLists := make([]MailingList, len(listNames))
	for i, name := range listNames {
		mailingLists[i] = MailingList{Name: name}
	}

	record := &SentNewsletter{
		Subject:      subject,
		SenderName:   senderName,
		RawMarkdown:  rawMarkdown,
		Recipients:   recipients,
		MailingLists: mailingLists,
	}

	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to create sent newsletter",
			slog.String("subject", subject),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("create sent newsletter: %w", result.Error)
	}

	if err := r.db.WithContext(ctx).Preload("Recipients").Preload("MailingLists").First(record, record.ID).Error; err != nil {
		return nil, fmt.Errorf("loading sent newsletter associations: %w", err)
	}

	return ToDomainSentNewsletter(record), nil
}

func (r *MailingListRepository) GetAllSentNewsletters(ctx context.Context) ([]domain.SentNewsletter, error) {
	var records []SentNewsletter
	result := r.db.WithContext(ctx).Preload("MailingLists").Find(&records)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get sent newsletters", slog.Any("error", result.Error))
		return nil, fmt.Errorf("get all sent newsletters: %w", result.Error)
	}
	return ToDomainSentNewsletters(records), nil
}

func (r *MailingListRepository) GetSentNewsletterByID(ctx context.Context, id uint) (*domain.SentNewsletter, error) {
	var record SentNewsletter
	result := r.db.WithContext(ctx).Preload("Recipients").Preload("MailingLists").First(&record, id)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to get sent newsletter",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return nil, fmt.Errorf("get sent newsletter %d: %w", id, result.Error)
	}
	return ToDomainSentNewsletter(&record), nil
}

func (r *MailingListRepository) DeleteSentNewsletter(ctx context.Context, id uint) error {
	record := &SentNewsletter{}
	record.ID = id
	if err := r.db.WithContext(ctx).Select("Recipients", "MailingLists").Delete(record).Error; err != nil {
		r.logger.ErrorContext(ctx, "failed to delete sent newsletter",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", err),
		)
		return fmt.Errorf("delete sent newsletter %d: %w", id, err)
	}
	return nil
}
