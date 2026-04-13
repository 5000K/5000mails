package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

func confirmedUser(id uint, listID uint, email string) *domain.User {
	now := time.Now()
	return &domain.User{ID: id, MailingListID: listID, Email: email, Name: "Test", ConfirmedAt: &now}
}

func TestMailService_SendToList(t *testing.T) {
	metadata := domain.MailMetadata{Subject: "Hello", SenderName: "Bot"}
	list := &domain.MailingList{ID: 5, Name: "weekly"}

	t.Run("skips send when no confirmed recipients", func(t *testing.T) {
		sender := &fakeSender{}
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(),
			&fakeRenderer{metadata: metadata, body: "body"},
			sender,
		)
		if err := svc.SendToList(context.Background(), "weekly", "# Hi", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sender.calls) != 0 {
			t.Errorf("expected no send calls, got %d", len(sender.calls))
		}
	})

	t.Run("renders and sends to confirmed recipients", func(t *testing.T) {
		users := newFakeUserRepo(
			confirmedUser(1, 5, "alice@example.com"),
			confirmedUser(2, 5, "bob@example.com"),
		)
		sender := &fakeSender{}
		svc := NewMailService(newFakeListRepo(list), users, &fakeRenderer{metadata: metadata, body: "rendered"}, sender)

		if err := svc.SendToList(context.Background(), "weekly", "# Hi", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sender.calls) != 1 {
			t.Fatalf("expected 1 send call, got %d", len(sender.calls))
		}
		call := sender.calls[0]
		if len(call.recipients) != 2 {
			t.Errorf("expected 2 recipients, got %d", len(call.recipients))
		}
		if call.body != "rendered" {
			t.Errorf("expected body %q, got %q", "rendered", call.body)
		}
		if call.metadata != metadata {
			t.Errorf("expected metadata %+v, got %+v", metadata, call.metadata)
		}
	})

	t.Run("wraps GetListByName error", func(t *testing.T) {
		listRepo := newFakeListRepo()
		listRepo.getByNameErr = errors.New("list missing")
		svc := NewMailService(listRepo, newFakeUserRepo(), &fakeRenderer{}, &fakeSender{})
		err := svc.SendToList(context.Background(), "ghost", "raw", nil)
		if !errors.Is(err, listRepo.getByNameErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("wraps GetConfirmedUsers error", func(t *testing.T) {
		userRepo := newFakeUserRepo()
		userRepo.getConfirmedErr = errors.New("db down")
		svc := NewMailService(newFakeListRepo(list), userRepo, &fakeRenderer{}, &fakeSender{})
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, userRepo.getConfirmedErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("wraps renderer error", func(t *testing.T) {
		renderErr := errors.New("template broken")
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(confirmedUser(1, 5, "a@example.com")),
			&fakeRenderer{err: renderErr},
			&fakeSender{},
		)
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, renderErr) {
			t.Errorf("expected wrapped render error, got: %v", err)
		}
	})

	t.Run("wraps sender error", func(t *testing.T) {
		sendErr := errors.New("smtp refused")
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(confirmedUser(1, 5, "a@example.com")),
			&fakeRenderer{metadata: metadata, body: "body"},
			&fakeSender{err: sendErr},
		)
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, sendErr) {
			t.Errorf("expected wrapped send error, got: %v", err)
		}
	})
}

func TestMailService_SendTestMail(t *testing.T) {
	metadata := domain.MailMetadata{Subject: "Test", SenderName: "Bot"}
	recipient := domain.User{ID: 1, Email: "dev@example.com", Name: "Dev"}

	t.Run("renders and sends to given recipient", func(t *testing.T) {
		sender := &fakeSender{}
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), &fakeRenderer{metadata: metadata, body: "preview"}, sender)

		if err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sender.calls) != 1 {
			t.Fatalf("expected 1 send call, got %d", len(sender.calls))
		}
		call := sender.calls[0]
		if len(call.recipients) != 1 || call.recipients[0].Email != recipient.Email {
			t.Errorf("unexpected recipients: %+v", call.recipients)
		}
		if call.body != "preview" {
			t.Errorf("expected body %q, got %q", "preview", call.body)
		}
	})

	t.Run("wraps renderer error", func(t *testing.T) {
		renderErr := errors.New("bad template")
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), &fakeRenderer{err: renderErr}, &fakeSender{})
		err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil)
		if !errors.Is(err, renderErr) {
			t.Errorf("expected wrapped render error, got: %v", err)
		}
	})

	t.Run("wraps sender error", func(t *testing.T) {
		sendErr := errors.New("smtp gone")
		svc := NewMailService(
			newFakeListRepo(),
			newFakeUserRepo(),
			&fakeRenderer{metadata: metadata, body: "body"},
			&fakeSender{err: sendErr},
		)
		err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil)
		if !errors.Is(err, sendErr) {
			t.Errorf("expected wrapped send error, got: %v", err)
		}
	})
}
