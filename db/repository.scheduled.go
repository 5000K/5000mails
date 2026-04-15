package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/5000K/5000mails/domain"
)

func (r *MailingListRepository) CreateScheduledMail(ctx context.Context, mailingListName, rawMarkdown string, scheduledAt int64, topicNames []string) (*domain.ScheduledMail, error) {
	topics := make([]ScheduledMailTopic, len(topicNames))
	for i, name := range topicNames {
		topics[i] = ScheduledMailTopic{TopicName: name}
	}
	m := &ScheduledMail{
		MailingListName: mailingListName,
		RawMarkdown:     rawMarkdown,
		ScheduledAt:     scheduledAt,
		Topics:          topics,
	}
	result := r.db.WithContext(ctx).Create(m)
	if result.Error != nil {
		return nil, fmt.Errorf("creating scheduled mail for list %q: %w", mailingListName, result.Error)
	}
	return ToDomainScheduledMail(m), nil
}

func (r *MailingListRepository) GetAllScheduledMails(ctx context.Context) ([]domain.ScheduledMail, error) {
	var mails []ScheduledMail
	result := r.db.WithContext(ctx).Preload("Topics").Order("scheduled_at asc").Find(&mails)
	if result.Error != nil {
		return nil, fmt.Errorf("listing scheduled mails: %w", result.Error)
	}
	return ToDomainScheduledMails(mails), nil
}

func (r *MailingListRepository) GetScheduledMailByID(ctx context.Context, id uint) (*domain.ScheduledMail, error) {
	var m ScheduledMail
	result := r.db.WithContext(ctx).Preload("Topics").First(&m, id)
	if result.Error != nil {
		return nil, fmt.Errorf("getting scheduled mail %d: %w", id, result.Error)
	}
	return ToDomainScheduledMail(&m), nil
}

func (r *MailingListRepository) GetPendingScheduledMails(ctx context.Context, now int64) ([]domain.ScheduledMail, error) {
	var mails []ScheduledMail
	result := r.db.WithContext(ctx).
		Preload("Topics").
		Where("scheduled_at <= ? AND sent_at IS NULL", now).
		Order("scheduled_at asc").
		Find(&mails)
	if result.Error != nil {
		return nil, fmt.Errorf("getting pending scheduled mails: %w", result.Error)
	}
	return ToDomainScheduledMails(mails), nil
}

func (r *MailingListRepository) UpdateScheduledMailTime(ctx context.Context, id uint, scheduledAt int64) (*domain.ScheduledMail, error) {
	result := r.db.WithContext(ctx).Model(&ScheduledMail{}).Where("id = ?", id).Update("scheduled_at", scheduledAt)
	if result.Error != nil {
		return nil, fmt.Errorf("rescheduling mail %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("rescheduling mail %d: not found", id)
	}
	return r.GetScheduledMailByID(ctx, id)
}

func (r *MailingListRepository) UpdateScheduledMailContent(ctx context.Context, id uint, rawMarkdown string) (*domain.ScheduledMail, error) {
	result := r.db.WithContext(ctx).Model(&ScheduledMail{}).Where("id = ?", id).Update("raw_markdown", rawMarkdown)
	if result.Error != nil {
		return nil, fmt.Errorf("updating content of scheduled mail %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("updating content of scheduled mail %d: not found", id)
	}
	return r.GetScheduledMailByID(ctx, id)
}

func (r *MailingListRepository) MarkScheduledMailSent(ctx context.Context, id uint, sentAt int64) error {
	result := r.db.WithContext(ctx).Model(&ScheduledMail{}).Where("id = ?", id).Update("sent_at", sentAt)
	if result.Error != nil {
		r.logger.ErrorContext(ctx, "failed to mark scheduled mail as sent",
			slog.Uint64("id", uint64(id)),
			slog.Any("error", result.Error),
		)
		return fmt.Errorf("marking scheduled mail %d as sent: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("marking scheduled mail %d as sent: not found", id)
	}
	return nil
}

func (r *MailingListRepository) DeleteScheduledMail(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&ScheduledMail{}, id)
	if result.Error != nil {
		return fmt.Errorf("deleting scheduled mail %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("deleting scheduled mail %d: not found", id)
	}
	return nil
}
