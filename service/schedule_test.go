package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

// fakeScheduledMailRepo is an in-memory ScheduledMailRepository.
type fakeScheduledMailRepo struct {
	mu     sync.Mutex
	mails  map[uint]*domain.ScheduledMail
	nextID uint

	createErr  error
	getAllErr  error
	getErr     error
	pendingErr error
	updateErr  error
	markErr    error
	deleteErr  error
}

func newFakeScheduledMailRepo(seed ...*domain.ScheduledMail) *fakeScheduledMailRepo {
	r := &fakeScheduledMailRepo{mails: make(map[uint]*domain.ScheduledMail), nextID: 1}
	for _, m := range seed {
		r.mails[m.ID] = m
		if m.ID >= r.nextID {
			r.nextID = m.ID + 1
		}
	}
	return r
}

func (r *fakeScheduledMailRepo) CreateScheduledMail(_ context.Context, mailingListName, rawMarkdown string, scheduledAt int64) (*domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return nil, r.createErr
	}
	m := &domain.ScheduledMail{ID: r.nextID, MailingListName: mailingListName, RawMarkdown: rawMarkdown, ScheduledAt: scheduledAt}
	r.nextID++
	r.mails[m.ID] = m
	return m, nil
}

func (r *fakeScheduledMailRepo) GetAllScheduledMails(_ context.Context) ([]domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getAllErr != nil {
		return nil, r.getAllErr
	}
	out := make([]domain.ScheduledMail, 0, len(r.mails))
	for _, m := range r.mails {
		out = append(out, *m)
	}
	return out, nil
}

func (r *fakeScheduledMailRepo) GetScheduledMailByID(_ context.Context, id uint) (*domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getErr != nil {
		return nil, r.getErr
	}
	m, ok := r.mails[id]
	if !ok {
		return nil, fmt.Errorf("scheduled mail %d not found", id)
	}
	return m, nil
}

func (r *fakeScheduledMailRepo) GetPendingScheduledMails(_ context.Context, now int64) ([]domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pendingErr != nil {
		return nil, r.pendingErr
	}
	var out []domain.ScheduledMail
	for _, m := range r.mails {
		if m.ScheduledAt <= now && m.SentAt == nil {
			out = append(out, *m)
		}
	}
	return out, nil
}

func (r *fakeScheduledMailRepo) UpdateScheduledMailTime(_ context.Context, id uint, scheduledAt int64) (*domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	m, ok := r.mails[id]
	if !ok {
		return nil, fmt.Errorf("scheduled mail %d not found", id)
	}
	m.ScheduledAt = scheduledAt
	return m, nil
}

func (r *fakeScheduledMailRepo) UpdateScheduledMailContent(_ context.Context, id uint, rawMarkdown string) (*domain.ScheduledMail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	m, ok := r.mails[id]
	if !ok {
		return nil, fmt.Errorf("scheduled mail %d not found", id)
	}
	m.RawMarkdown = rawMarkdown
	return m, nil
}

func (r *fakeScheduledMailRepo) MarkScheduledMailSent(_ context.Context, id uint, sentAt int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.markErr != nil {
		return r.markErr
	}
	m, ok := r.mails[id]
	if !ok {
		return fmt.Errorf("scheduled mail %d not found", id)
	}
	m.SentAt = &sentAt
	return nil
}

func (r *fakeScheduledMailRepo) DeleteScheduledMail(_ context.Context, id uint) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.mails[id]; !ok {
		return fmt.Errorf("scheduled mail %d not found", id)
	}
	delete(r.mails, id)
	return nil
}

// fakeListMailSender records SendToList calls.
type fakeListMailSender struct {
	mu    sync.Mutex
	calls []listSendCall
	err   error
}

type listSendCall struct {
	listName string
	raw      string
}

func (s *fakeListMailSender) SendToList(_ context.Context, listName string, raw string, _ map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.calls = append(s.calls, listSendCall{listName: listName, raw: raw})
	return nil
}

func (s *fakeListMailSender) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

// --- tests ---

func TestSchedulingService_Schedule(t *testing.T) {
	repo := newFakeScheduledMailRepo()
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	m, err := svc.Schedule(context.Background(), "newsletter", "# Hello", 1_000_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.MailingListName != "newsletter" {
		t.Errorf("want list newsletter, got %q", m.MailingListName)
	}
	if m.ScheduledAt != 1_000_000 {
		t.Errorf("want scheduledAt 1000000, got %d", m.ScheduledAt)
	}
	if m.SentAt != nil {
		t.Error("new scheduled mail should not be marked as sent")
	}
}

func TestSchedulingService_Schedule_RepoError(t *testing.T) {
	repo := newFakeScheduledMailRepo()
	repo.createErr = fmt.Errorf("db error")
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	_, err := svc.Schedule(context.Background(), "newsletter", "# Hello", 1_000_000)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSchedulingService_List(t *testing.T) {
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "a", RawMarkdown: "x", ScheduledAt: 100},
		&domain.ScheduledMail{ID: 2, MailingListName: "b", RawMarkdown: "y", ScheduledAt: 200},
	)
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	mails, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mails) != 2 {
		t.Errorf("want 2 mails, got %d", len(mails))
	}
}

func TestSchedulingService_Get(t *testing.T) {
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "x", RawMarkdown: "raw", ScheduledAt: 42},
	)
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	m, err := svc.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ID != 1 || m.RawMarkdown != "raw" {
		t.Errorf("unexpected mail: %+v", m)
	}
}

func TestSchedulingService_Get_NotFound(t *testing.T) {
	svc := newTestSchedulingService(newFakeScheduledMailRepo(), &fakeListMailSender{})
	_, err := svc.Get(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestSchedulingService_Delete(t *testing.T) {
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "x", RawMarkdown: "raw", ScheduledAt: 42},
	)
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mails, _ := repo.GetAllScheduledMails(context.Background())
	if len(mails) != 0 {
		t.Error("mail should have been deleted")
	}
}

func TestSchedulingService_Reschedule(t *testing.T) {
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "x", RawMarkdown: "raw", ScheduledAt: 100},
	)
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	m, err := svc.Reschedule(context.Background(), 1, 9999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.ScheduledAt != 9999 {
		t.Errorf("want scheduledAt 9999, got %d", m.ScheduledAt)
	}
}

func TestSchedulingService_ReplaceContent(t *testing.T) {
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "x", RawMarkdown: "old", ScheduledAt: 100},
	)
	svc := newTestSchedulingService(repo, &fakeListMailSender{})

	m, err := svc.ReplaceContent(context.Background(), 1, "# New content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.RawMarkdown != "# New content" {
		t.Errorf("content not replaced, got %q", m.RawMarkdown)
	}
}

func TestSchedulingService_DispatchesDueMails(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).Unix()
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "newsletter", RawMarkdown: "# Hi", ScheduledAt: past},
	)
	mailer := &fakeListMailSender{}
	svc := newTestSchedulingService(repo, mailer)

	svc.dispatchDue()

	if mailer.callCount() != 1 {
		t.Fatalf("expected 1 send call, got %d", mailer.callCount())
	}
	if mailer.calls[0].listName != "newsletter" {
		t.Errorf("unexpected list name: %s", mailer.calls[0].listName)
	}

	m, _ := repo.GetScheduledMailByID(context.Background(), 1)
	if m.SentAt == nil {
		t.Error("mail should be marked as sent after dispatch")
	}
}

func TestSchedulingService_DoesNotDispatchFutureMails(t *testing.T) {
	future := time.Now().Add(1 * time.Hour).Unix()
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "newsletter", RawMarkdown: "# Hi", ScheduledAt: future},
	)
	mailer := &fakeListMailSender{}
	svc := newTestSchedulingService(repo, mailer)

	svc.dispatchDue()

	if mailer.callCount() != 0 {
		t.Errorf("expected no send calls for future mail, got %d", mailer.callCount())
	}
}

func TestSchedulingService_DoesNotRedispatchSentMails(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).Unix()
	sentAt := past + 10
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "newsletter", RawMarkdown: "# Hi", ScheduledAt: past, SentAt: &sentAt},
	)
	mailer := &fakeListMailSender{}
	svc := newTestSchedulingService(repo, mailer)

	svc.dispatchDue()

	if mailer.callCount() != 0 {
		t.Errorf("expected no send calls for already-sent mail, got %d", mailer.callCount())
	}
}

func TestSchedulingService_SendErrorDoesNotMarkAsSent(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour).Unix()
	repo := newFakeScheduledMailRepo(
		&domain.ScheduledMail{ID: 1, MailingListName: "newsletter", RawMarkdown: "# Hi", ScheduledAt: past},
	)
	mailer := &fakeListMailSender{err: fmt.Errorf("smtp failure")}
	svc := newTestSchedulingService(repo, mailer)

	svc.dispatchDue()

	m, _ := repo.GetScheduledMailByID(context.Background(), 1)
	if m.SentAt != nil {
		t.Error("failed mail should not be marked as sent")
	}
}

func newTestSchedulingService(repo *fakeScheduledMailRepo, mailer *fakeListMailSender) *SchedulingService {
	return NewSchedulingService(repo, mailer, time.Minute, noopLogger())
}

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
