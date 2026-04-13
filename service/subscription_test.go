package service

import (
	"context"
	"errors"
	"testing"

	"github.com/5000K/5000mails/domain"
)

func newSubscriptionSvc(
	lists *fakeListRepo,
	users *fakeUserRepo,
	confs *fakeConfirmationRepo,
	renderer *fakeRenderer,
	sender *fakeSender,
) *SubscriptionService {
	return NewSubscriptionService(lists, users, confs, renderer, sender, "# Confirm your subscription\nToken: {{.token}}")
}

func TestSubscriptionService_Subscribe(t *testing.T) {
	metadata := domain.MailMetadata{Subject: "Confirm", SenderName: "Bot"}
	list := &domain.MailingList{ID: 1, Name: "weekly"}

	t.Run("adds user, creates confirmation, sends mail", func(t *testing.T) {
		users := newFakeUserRepo()
		confs := newFakeConfirmationRepo()
		sender := &fakeSender{}
		svc := newSubscriptionSvc(newFakeListRepo(list), users, confs, &fakeRenderer{metadata: metadata, body: "click here"}, sender)

		user, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.Email != "alice@example.com" {
			t.Errorf("unexpected user email: %s", user.Email)
		}
		if len(confs.confirmations) != 1 {
			t.Errorf("expected 1 confirmation, got %d", len(confs.confirmations))
		}
		if len(sender.calls) != 1 {
			t.Fatalf("expected 1 send call, got %d", len(sender.calls))
		}
		if sender.calls[0].recipient.Email != "alice@example.com" {
			t.Errorf("unexpected recipient: %+v", sender.calls[0].recipient)
		}
	})

	t.Run("passes Recipient in render data", func(t *testing.T) {
		renderer := &fakeRenderer{metadata: metadata, body: "click here"}
		svc := newSubscriptionSvc(newFakeListRepo(list), newFakeUserRepo(), newFakeConfirmationRepo(), renderer, &fakeSender{})

		user, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := renderer.lastData["Recipient"]
		if !ok {
			t.Fatal("expected Recipient key in render data")
		}
		if u, ok := got.(domain.User); !ok || u.Email != user.Email {
			t.Errorf("unexpected Recipient in render data: %+v", got)
		}
	})

	t.Run("returns error when GetListByName fails", func(t *testing.T) {
		repo := newFakeListRepo()
		repo.getByNameErr = errors.New("list missing")
		svc := newSubscriptionSvc(repo, newFakeUserRepo(), newFakeConfirmationRepo(), &fakeRenderer{}, &fakeSender{})
		_, err := svc.Subscribe(context.Background(), "ghost", "Bob", "bob@example.com")
		if !errors.Is(err, repo.getByNameErr) {
			t.Errorf("expected wrapped list error, got: %v", err)
		}
	})

	t.Run("returns error when AddUser fails", func(t *testing.T) {
		users := newFakeUserRepo()
		users.addErr = errors.New("duplicate email")
		svc := newSubscriptionSvc(newFakeListRepo(list), users, newFakeConfirmationRepo(), &fakeRenderer{}, &fakeSender{})
		_, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if !errors.Is(err, users.addErr) {
			t.Errorf("expected wrapped add error, got: %v", err)
		}
	})

	t.Run("returns error when CreateConfirmation fails", func(t *testing.T) {
		confs := newFakeConfirmationRepo()
		confs.createErr = errors.New("db full")
		svc := newSubscriptionSvc(newFakeListRepo(list), newFakeUserRepo(), confs, &fakeRenderer{}, &fakeSender{})
		_, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if !errors.Is(err, confs.createErr) {
			t.Errorf("expected wrapped confirmation error, got: %v", err)
		}
	})

	t.Run("returns error when Render fails", func(t *testing.T) {
		renderErr := errors.New("bad template")
		svc := newSubscriptionSvc(newFakeListRepo(list), newFakeUserRepo(), newFakeConfirmationRepo(), &fakeRenderer{err: renderErr}, &fakeSender{})
		_, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if !errors.Is(err, renderErr) {
			t.Errorf("expected wrapped render error, got: %v", err)
		}
	})

	t.Run("returns error when SendMail fails", func(t *testing.T) {
		sendErr := errors.New("smtp gone")
		metadata := domain.MailMetadata{Subject: "Confirm"}
		svc := newSubscriptionSvc(
			newFakeListRepo(list),
			newFakeUserRepo(),
			newFakeConfirmationRepo(),
			&fakeRenderer{metadata: metadata, body: "ok"},
			&fakeSender{err: sendErr},
		)
		_, err := svc.Subscribe(context.Background(), "weekly", "Alice", "alice@example.com")
		if !errors.Is(err, sendErr) {
			t.Errorf("expected wrapped send error, got: %v", err)
		}
	})
}

func TestSubscriptionService_Confirm(t *testing.T) {
	t.Run("confirms user and deletes confirmation", func(t *testing.T) {
		users := newFakeUserRepo(&domain.User{ID: 1, Email: "alice@example.com", MailingListID: 1})
		confs := newFakeConfirmationRepo(&domain.Confirmation{ID: 1, UserID: 1, Token: "abc123"})
		svc := newSubscriptionSvc(newFakeListRepo(), users, confs, &fakeRenderer{}, &fakeSender{})

		if err := svc.Confirm(context.Background(), "abc123"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !users.users[1].IsConfirmed() {
			t.Error("expected user to be confirmed")
		}
		if len(confs.confirmations) != 0 {
			t.Errorf("expected confirmation to be deleted, got %d remaining", len(confs.confirmations))
		}
	})

	t.Run("returns error for unknown token", func(t *testing.T) {
		confs := newFakeConfirmationRepo()
		confs.getErr = errors.New("token not found")
		svc := newSubscriptionSvc(newFakeListRepo(), newFakeUserRepo(), confs, &fakeRenderer{}, &fakeSender{})
		err := svc.Confirm(context.Background(), "bad-token")
		if !errors.Is(err, confs.getErr) {
			t.Errorf("expected wrapped token error, got: %v", err)
		}
	})

	t.Run("returns error when ConfirmUser fails", func(t *testing.T) {
		users := newFakeUserRepo()
		users.confirmErr = errors.New("confirm failed")
		confs := newFakeConfirmationRepo(&domain.Confirmation{ID: 1, UserID: 99, Token: "tok"})
		svc := newSubscriptionSvc(newFakeListRepo(), users, confs, &fakeRenderer{}, &fakeSender{})
		err := svc.Confirm(context.Background(), "tok")
		if !errors.Is(err, users.confirmErr) {
			t.Errorf("expected wrapped confirm error, got: %v", err)
		}
	})

	t.Run("returns error when DeleteConfirmation fails", func(t *testing.T) {
		users := newFakeUserRepo(&domain.User{ID: 1, Email: "alice@example.com", MailingListID: 1})
		confs := newFakeConfirmationRepo(&domain.Confirmation{ID: 1, UserID: 1, Token: "tok"})
		confs.deleteErr = errors.New("delete failed")
		svc := newSubscriptionSvc(newFakeListRepo(), users, confs, &fakeRenderer{}, &fakeSender{})
		err := svc.Confirm(context.Background(), "tok")
		if !errors.Is(err, confs.deleteErr) {
			t.Errorf("expected wrapped delete error, got: %v", err)
		}
	})
}

func TestSubscriptionService_Unsubscribe(t *testing.T) {
	u := &domain.User{ID: 1, Email: "alice@example.com", MailingListID: 1, UnsubscribeToken: "tok-alice"}

	t.Run("removes user by unsubscribe token", func(t *testing.T) {
		users := newFakeUserRepo(u)
		svc := newSubscriptionSvc(newFakeListRepo(), users, newFakeConfirmationRepo(), &fakeRenderer{}, &fakeSender{})
		if err := svc.Unsubscribe(context.Background(), "tok-alice"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := users.users[1]; exists {
			t.Error("expected user to be removed")
		}
	})

	t.Run("returns error when token not found", func(t *testing.T) {
		users := newFakeUserRepo()
		users.getByUnsubscribeTokenErr = errors.New("not found")
		svc := newSubscriptionSvc(newFakeListRepo(), users, newFakeConfirmationRepo(), &fakeRenderer{}, &fakeSender{})
		err := svc.Unsubscribe(context.Background(), "bad-token")
		if !errors.Is(err, users.getByUnsubscribeTokenErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("returns error when RemoveUser fails", func(t *testing.T) {
		users := newFakeUserRepo(u)
		users.removeErr = errors.New("delete failed")
		svc := newSubscriptionSvc(newFakeListRepo(), users, newFakeConfirmationRepo(), &fakeRenderer{}, &fakeSender{})
		err := svc.Unsubscribe(context.Background(), "tok-alice")
		if !errors.Is(err, users.removeErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})
}
