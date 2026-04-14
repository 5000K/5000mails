package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

func confirmedUser(id uint, listName string, email string) *domain.User {
	now := time.Now()
	return &domain.User{ID: id, MailingListName: listName, Email: email, Name: "Test", ConfirmedAt: &now}
}

func TestMailService_SendToList(t *testing.T) {
	metadata := domain.MailMetadata{Subject: "Hello", SenderName: "Bot"}
	list := &domain.MailingList{Name: "weekly"}

	t.Run("skips send when no confirmed recipients", func(t *testing.T) {
		sender := &fakeSender{}
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(),
			newFakeNewsletterRepo(),
			&fakeRenderer{metadata: metadata, body: "body"},
			sender,
			"https://example.com",
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
			confirmedUser(1, "weekly", "alice@example.com"),
			confirmedUser(2, "weekly", "bob@example.com"),
		)
		sender := &fakeSender{}
		svc := NewMailService(newFakeListRepo(list), users, newFakeNewsletterRepo(), &fakeRenderer{metadata: metadata, body: "rendered"}, sender, "https://example.com")

		if err := svc.SendToList(context.Background(), "weekly", "# Hi", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sender.calls) != 2 {
			t.Fatalf("expected 2 send calls, got %d", len(sender.calls))
		}
		emails := map[string]bool{}
		for _, call := range sender.calls {
			emails[call.recipient.Email] = true
			if call.body != "rendered" {
				t.Errorf("expected body %q, got %q", "rendered", call.body)
			}
			if call.metadata != metadata {
				t.Errorf("expected metadata %+v, got %+v", metadata, call.metadata)
			}
		}
		if !emails["alice@example.com"] || !emails["bob@example.com"] {
			t.Errorf("expected both recipients to receive mail, got: %v", emails)
		}
	})

	t.Run("injects Recipient into render data per recipient", func(t *testing.T) {
		user := confirmedUser(1, "weekly", "alice@example.com")
		renderer := &fakeRenderer{metadata: metadata, body: "body"}
		svc := NewMailService(newFakeListRepo(list), newFakeUserRepo(user), newFakeNewsletterRepo(), renderer, &fakeSender{}, "https://example.com")

		if err := svc.SendToList(context.Background(), "weekly", "raw", nil); err != nil {
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

	t.Run("injects unsubscribeURL into render data per recipient", func(t *testing.T) {
		user := confirmedUser(1, "weekly", "alice@example.com")
		user.UnsubscribeToken = "unsub-tok"
		renderer := &fakeRenderer{metadata: metadata, body: "body"}
		svc := NewMailService(newFakeListRepo(list), newFakeUserRepo(user), newFakeNewsletterRepo(), renderer, &fakeSender{}, "https://example.com")

		if err := svc.SendToList(context.Background(), "weekly", "raw", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantURL := "https://example.com/unsubscribe/unsub-tok"
		if got, _ := renderer.lastData["unsubscribeURL"].(string); got != wantURL {
			t.Errorf("unsubscribeURL = %q, want %q", got, wantURL)
		}
	})

	t.Run("wraps GetListByName error", func(t *testing.T) {
		listRepo := newFakeListRepo()
		listRepo.getByNameErr = errors.New("list missing")
		svc := NewMailService(listRepo, newFakeUserRepo(), newFakeNewsletterRepo(), &fakeRenderer{}, &fakeSender{}, "https://example.com")
		err := svc.SendToList(context.Background(), "ghost", "raw", nil)
		if !errors.Is(err, listRepo.getByNameErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("wraps GetConfirmedUsers error", func(t *testing.T) {
		userRepo := newFakeUserRepo()
		userRepo.getConfirmedErr = errors.New("db down")
		svc := NewMailService(newFakeListRepo(list), userRepo, newFakeNewsletterRepo(), &fakeRenderer{}, &fakeSender{}, "https://example.com")
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, userRepo.getConfirmedErr) {
			t.Errorf("expected wrapped error, got: %v", err)
		}
	})

	t.Run("wraps renderer error", func(t *testing.T) {
		renderErr := errors.New("template broken")
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(confirmedUser(1, "weekly", "a@example.com")),
			newFakeNewsletterRepo(),
			&fakeRenderer{err: renderErr},
			&fakeSender{},
			"https://example.com",
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
			newFakeUserRepo(confirmedUser(1, "weekly", "a@example.com")),
			newFakeNewsletterRepo(),
			&fakeRenderer{metadata: metadata, body: "body"},
			&fakeSender{err: sendErr},
			"https://example.com",
		)
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, sendErr) {
			t.Errorf("expected wrapped send error, got: %v", err)
		}
	})

	t.Run("archives newsletter after sending", func(t *testing.T) {
		users := newFakeUserRepo(
			confirmedUser(1, "weekly", "alice@example.com"),
			confirmedUser(2, "weekly", "bob@example.com"),
		)
		newsletterRepo := newFakeNewsletterRepo()
		svc := NewMailService(newFakeListRepo(list), users, newsletterRepo, &fakeRenderer{metadata: metadata, body: "body"}, &fakeSender{}, "https://example.com")

		if err := svc.SendToList(context.Background(), "weekly", "# Raw", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(newsletterRepo.newsletters) != 1 {
			t.Fatalf("expected 1 archived newsletter, got %d", len(newsletterRepo.newsletters))
		}
		for _, n := range newsletterRepo.newsletters {
			if n.Subject != metadata.Subject {
				t.Errorf("expected subject %q, got %q", metadata.Subject, n.Subject)
			}
			if n.RawMarkdown != "# Raw" {
				t.Errorf("expected raw %q, got %q", "# Raw", n.RawMarkdown)
			}
			if len(n.Recipients) != 2 {
				t.Errorf("expected 2 recipients, got %d", len(n.Recipients))
			}
			if len(n.MailingLists) != 1 || n.MailingLists[0].Name != "weekly" {
				t.Errorf("unexpected mailing lists: %+v", n.MailingLists)
			}
		}
	})

	t.Run("wraps newsletter archive error", func(t *testing.T) {
		archiveErr := errors.New("archive failed")
		newsletterRepo := newFakeNewsletterRepo()
		newsletterRepo.createErr = archiveErr
		svc := NewMailService(
			newFakeListRepo(list),
			newFakeUserRepo(confirmedUser(1, "weekly", "a@example.com")),
			newsletterRepo,
			&fakeRenderer{metadata: metadata, body: "body"},
			&fakeSender{},
			"https://example.com",
		)
		err := svc.SendToList(context.Background(), "weekly", "raw", nil)
		if !errors.Is(err, archiveErr) {
			t.Errorf("expected wrapped archive error, got: %v", err)
		}
	})
}

func TestMailService_SendTestMail(t *testing.T) {
	metadata := domain.MailMetadata{Subject: "Test", SenderName: "Bot"}
	recipient := domain.User{ID: 1, Email: "dev@example.com", Name: "Dev"}

	t.Run("renders and sends to given recipient", func(t *testing.T) {
		sender := &fakeSender{}
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), newFakeNewsletterRepo(), &fakeRenderer{metadata: metadata, body: "preview"}, sender, "https://example.com")

		if err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sender.calls) != 1 {
			t.Fatalf("expected 1 send call, got %d", len(sender.calls))
		}
		call := sender.calls[0]
		if call.recipient.Email != recipient.Email {
			t.Errorf("unexpected recipient: %+v", call.recipient)
		}
		if call.body != "preview" {
			t.Errorf("expected body %q, got %q", "preview", call.body)
		}
	})

	t.Run("injects Recipient into render data", func(t *testing.T) {
		renderer := &fakeRenderer{metadata: metadata, body: "body"}
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), newFakeNewsletterRepo(), renderer, &fakeSender{}, "https://example.com")

		if err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, ok := renderer.lastData["Recipient"]
		if !ok {
			t.Fatal("expected Recipient key in render data")
		}
		if u, ok := got.(domain.User); !ok || u.Email != recipient.Email {
			t.Errorf("unexpected Recipient in render data: %+v", got)
		}
	})

	t.Run("injects unsubscribeURL into render data", func(t *testing.T) {
		recipientWithToken := domain.User{ID: 1, Email: "dev@example.com", Name: "Dev", UnsubscribeToken: "my-unsub-tok"}
		renderer := &fakeRenderer{metadata: metadata, body: "body"}
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), newFakeNewsletterRepo(), renderer, &fakeSender{}, "https://example.com")

		if err := svc.SendTestMail(context.Background(), recipientWithToken, "# Draft", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantURL := "https://example.com/unsubscribe/my-unsub-tok"
		if got, _ := renderer.lastData["unsubscribeURL"].(string); got != wantURL {
			t.Errorf("unsubscribeURL = %q, want %q", got, wantURL)
		}
	})

	t.Run("wraps renderer error", func(t *testing.T) {
		renderErr := errors.New("bad template")
		svc := NewMailService(newFakeListRepo(), newFakeUserRepo(), newFakeNewsletterRepo(), &fakeRenderer{err: renderErr}, &fakeSender{}, "https://example.com")
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
			newFakeNewsletterRepo(),
			&fakeRenderer{metadata: metadata, body: "body"},
			&fakeSender{err: sendErr},
			"https://example.com",
		)
		err := svc.SendTestMail(context.Background(), recipient, "# Draft", nil)
		if !errors.Is(err, sendErr) {
			t.Errorf("expected wrapped send error, got: %v", err)
		}
	})
}
