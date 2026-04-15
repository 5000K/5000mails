package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/5000K/5000mails/domain"
)

type ListMailSender interface {
	SendToList(ctx context.Context, listName string, raw string, topicNames []string, data map[string]any) error
}

type SchedulingService struct {
	repo     domain.ScheduledMailRepository
	mailer   ListMailSender
	interval time.Duration
	logger   *slog.Logger
	mu       sync.Mutex
	stop     chan struct{}
}

func NewSchedulingService(repo domain.ScheduledMailRepository, mailer ListMailSender, interval time.Duration, logger *slog.Logger) *SchedulingService {
	return &SchedulingService{
		repo:     repo,
		mailer:   mailer,
		interval: interval,
		logger:   logger,
		stop:     make(chan struct{}),
	}
}

func (s *SchedulingService) Start() {
	go s.loop()
}

func (s *SchedulingService) Stop() {
	close(s.stop)
}

func (s *SchedulingService) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.dispatchDue()
		}
	}
}

func (s *SchedulingService) dispatchDue() {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	now := time.Now().Unix()

	pending, err := s.repo.GetPendingScheduledMails(ctx, now)
	if err != nil {
		s.logger.ErrorContext(ctx, "fetching pending scheduled mails", slog.Any("error", err))
		return
	}

	for _, m := range pending {
		if err := s.mailer.SendToList(ctx, m.MailingListName, m.RawMarkdown, m.TopicNames, nil); err != nil {
			s.logger.ErrorContext(ctx, "sending scheduled mail",
				slog.Uint64("id", uint64(m.ID)),
				slog.String("list", m.MailingListName),
				slog.Any("error", err),
			)
			continue
		}
		sentAt := time.Now().Unix()
		if err := s.repo.MarkScheduledMailSent(ctx, m.ID, sentAt); err != nil {
			s.logger.ErrorContext(ctx, "marking scheduled mail as sent",
				slog.Uint64("id", uint64(m.ID)),
				slog.Any("error", err),
			)
		}
	}
}

func (s *SchedulingService) Schedule(ctx context.Context, mailingListName, rawMarkdown string, scheduledAt int64, topicNames []string) (*domain.ScheduledMail, error) {
	m, err := s.repo.CreateScheduledMail(ctx, mailingListName, rawMarkdown, scheduledAt, topicNames)
	if err != nil {
		return nil, fmt.Errorf("scheduling mail for list %q: %w", mailingListName, err)
	}
	return m, nil
}

func (s *SchedulingService) List(ctx context.Context) ([]domain.ScheduledMail, error) {
	mails, err := s.repo.GetAllScheduledMails(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing scheduled mails: %w", err)
	}
	return mails, nil
}

func (s *SchedulingService) Get(ctx context.Context, id uint) (*domain.ScheduledMail, error) {
	m, err := s.repo.GetScheduledMailByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting scheduled mail %d: %w", id, err)
	}
	return m, nil
}

func (s *SchedulingService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.DeleteScheduledMail(ctx, id); err != nil {
		return fmt.Errorf("deleting scheduled mail %d: %w", id, err)
	}
	return nil
}

func (s *SchedulingService) Reschedule(ctx context.Context, id uint, scheduledAt int64) (*domain.ScheduledMail, error) {
	m, err := s.repo.UpdateScheduledMailTime(ctx, id, scheduledAt)
	if err != nil {
		return nil, fmt.Errorf("rescheduling mail %d: %w", id, err)
	}
	return m, nil
}

func (s *SchedulingService) ReplaceContent(ctx context.Context, id uint, rawMarkdown string) (*domain.ScheduledMail, error) {
	m, err := s.repo.UpdateScheduledMailContent(ctx, id, rawMarkdown)
	if err != nil {
		return nil, fmt.Errorf("replacing content of scheduled mail %d: %w", id, err)
	}
	return m, nil
}
