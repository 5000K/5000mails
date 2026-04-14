package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/5000K/5000mails/domain"
)

type fakeSubscriber struct {
	subscribeErr   error
	confirmeErr    error
	unsubscribeErr error

	lastListName   string
	lastToken      string
	lastEmail      string
	lastUnsubToken string
}

func (f *fakeSubscriber) Subscribe(_ context.Context, listName, _, email string) (*domain.User, error) {
	f.lastListName = listName
	f.lastEmail = email
	if f.subscribeErr != nil {
		return nil, f.subscribeErr
	}
	return &domain.User{ID: 1, Name: "Alice", Email: email}, nil
}

func (f *fakeSubscriber) Confirm(_ context.Context, token string) error {
	f.lastToken = token
	return f.confirmeErr
}

func (f *fakeSubscriber) Unsubscribe(_ context.Context, token string) error {
	f.lastUnsubToken = token
	return f.unsubscribeErr
}

type fakeNewsletterPreviewer struct {
	body string
	err  error

	lastID    uint
	lastToken string
}

func (f *fakeNewsletterPreviewer) RenderNewsletter(_ context.Context, id uint, token string) (string, error) {
	f.lastID = id
	f.lastToken = token
	return f.body, f.err
}

func newTestHandler(sub *fakeSubscriber) *PublicHandler {
	return NewPublicHandler(sub, &fakeNewsletterPreviewer{body: "<html/>", err: nil}, RedirectPages{}, slog.Default())
}

func newTestHandlerWithRedirects(sub *fakeSubscriber, redirects RedirectPages) *PublicHandler {
	return NewPublicHandler(sub, &fakeNewsletterPreviewer{body: "<html/>", err: nil}, redirects, slog.Default())
}

func TestHandleSubscribe(t *testing.T) {
	t.Run("returns 202 on success", func(t *testing.T) {
		sub := &fakeSubscriber{}
		h := newTestHandler(sub)

		body := `{"name":"Alice","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()

		h.handleSubscribe(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d", w.Code)
		}
		if sub.lastListName != "weekly" {
			t.Errorf("expected listName %q, got %q", "weekly", sub.lastListName)
		}
		if sub.lastEmail != "alice@example.com" {
			t.Errorf("expected email %q, got %q", "alice@example.com", sub.lastEmail)
		}
	})

	t.Run("returns 400 on missing fields", func(t *testing.T) {
		h := newTestHandler(&fakeSubscriber{})

		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(`{"name":"Alice"}`))
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()

		h.handleSubscribe(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid JSON", func(t *testing.T) {
		h := newTestHandler(&fakeSubscriber{})

		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(`not-json`))
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()

		h.handleSubscribe(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("accepts form data", func(t *testing.T) {
		sub := &fakeSubscriber{}
		h := newTestHandler(sub)

		form := url.Values{"name": {"Alice"}, "email": {"alice@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()

		h.handleSubscribe(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d", w.Code)
		}
		if sub.lastEmail != "alice@example.com" {
			t.Errorf("expected email %q, got %q", "alice@example.com", sub.lastEmail)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		sub := &fakeSubscriber{subscribeErr: errors.New("db down")}
		h := newTestHandler(sub)

		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()

		h.handleSubscribe(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestHandleConfirm(t *testing.T) {
	t.Run("returns 200 on success", func(t *testing.T) {
		sub := &fakeSubscriber{}
		h := newTestHandler(sub)

		req := httptest.NewRequest(http.MethodGet, "/confirm/abc123", nil)
		req.SetPathValue("token", "abc123")
		w := httptest.NewRecorder()

		h.handleConfirm(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if sub.lastToken != "abc123" {
			t.Errorf("expected token %q, got %q", "abc123", sub.lastToken)
		}
	})

	t.Run("returns 400 on invalid token", func(t *testing.T) {
		sub := &fakeSubscriber{confirmeErr: errors.New("token not found")}
		h := newTestHandler(sub)

		req := httptest.NewRequest(http.MethodGet, "/confirm/bad", nil)
		req.SetPathValue("token", "bad")
		w := httptest.NewRecorder()

		h.handleConfirm(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestHandleUnsubscribe(t *testing.T) {
	t.Run("returns 200 on success", func(t *testing.T) {
		sub := &fakeSubscriber{}
		h := newTestHandler(sub)

		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/tok123", nil)
		req.SetPathValue("token", "tok123")
		w := httptest.NewRecorder()

		h.handleUnsubscribe(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		if sub.lastUnsubToken != "tok123" {
			t.Errorf("expected token %q, got %q", "tok123", sub.lastUnsubToken)
		}
	})

	t.Run("returns 400 on invalid token", func(t *testing.T) {
		sub := &fakeSubscriber{unsubscribeErr: errors.New("token not found")}
		h := newTestHandler(sub)

		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/bad", nil)
		req.SetPathValue("token", "bad")
		w := httptest.NewRecorder()

		h.handleUnsubscribe(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestRoutes(t *testing.T) {
	sub := &fakeSubscriber{}
	h := newTestHandler(sub)
	mux := h.Routes()

	t.Run("POST /{listName}/subscribe is routed", func(t *testing.T) {
		form := url.Values{"name": {"Alice"}, "email": {"alice@example.com"}}
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusAccepted {
			t.Errorf("expected 202, got %d", w.Code)
		}
	})

	t.Run("GET /confirm/{token} is routed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/confirm/mytoken", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("GET /unsubscribe/{token} is routed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/sometoken", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})

	t.Run("response has application/json content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/sometoken", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if ct := w.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}
	})

	t.Run("response body is valid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/sometoken", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		var got map[string]string
		if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
			t.Errorf("expected valid JSON response: %v", err)
		}
	})
}

func TestRedirectPages(t *testing.T) {
	redirects := RedirectPages{
		SubscribeSuccess:   "https://example.com/subscribe/success",
		SubscribeError:     "https://example.com/subscribe/error",
		ConfirmSuccess:     "https://example.com/confirm/success",
		ConfirmError:       "https://example.com/confirm/error",
		UnsubscribeSuccess: "https://example.com/unsubscribe/success",
		UnsubscribeError:   "https://example.com/unsubscribe/error",
	}

	t.Run("subscribe success redirects", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{}, redirects)
		body := `{"name":"Alice","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()
		h.handleSubscribe(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.SubscribeSuccess {
			t.Errorf("expected Location %q, got %q", redirects.SubscribeSuccess, loc)
		}
	})

	t.Run("subscribe error redirects on bad request", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{}, redirects)
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(`{"name":"Alice"}`))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()
		h.handleSubscribe(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.SubscribeError {
			t.Errorf("expected Location %q, got %q", redirects.SubscribeError, loc)
		}
	})

	t.Run("subscribe error redirects on service error", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{subscribeErr: errors.New("db down")}, redirects)
		body := `{"name":"Alice","email":"alice@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/weekly/subscribe", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("listName", "weekly")
		w := httptest.NewRecorder()
		h.handleSubscribe(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.SubscribeError {
			t.Errorf("expected Location %q, got %q", redirects.SubscribeError, loc)
		}
	})

	t.Run("confirm success redirects", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{}, redirects)
		req := httptest.NewRequest(http.MethodGet, "/confirm/abc123", nil)
		req.SetPathValue("token", "abc123")
		w := httptest.NewRecorder()
		h.handleConfirm(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.ConfirmSuccess {
			t.Errorf("expected Location %q, got %q", redirects.ConfirmSuccess, loc)
		}
	})

	t.Run("confirm error redirects", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{confirmeErr: errors.New("bad token")}, redirects)
		req := httptest.NewRequest(http.MethodGet, "/confirm/bad", nil)
		req.SetPathValue("token", "bad")
		w := httptest.NewRecorder()
		h.handleConfirm(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.ConfirmError {
			t.Errorf("expected Location %q, got %q", redirects.ConfirmError, loc)
		}
	})

	t.Run("unsubscribe success redirects", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{}, redirects)
		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/tok123", nil)
		req.SetPathValue("token", "tok123")
		w := httptest.NewRecorder()
		h.handleUnsubscribe(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.UnsubscribeSuccess {
			t.Errorf("expected Location %q, got %q", redirects.UnsubscribeSuccess, loc)
		}
	})

	t.Run("unsubscribe error redirects", func(t *testing.T) {
		h := newTestHandlerWithRedirects(&fakeSubscriber{unsubscribeErr: errors.New("bad token")}, redirects)
		req := httptest.NewRequest(http.MethodGet, "/unsubscribe/bad", nil)
		req.SetPathValue("token", "bad")
		w := httptest.NewRecorder()
		h.handleUnsubscribe(w, req)
		if w.Code != http.StatusSeeOther {
			t.Errorf("expected 303, got %d", w.Code)
		}
		if loc := w.Header().Get("Location"); loc != redirects.UnsubscribeError {
			t.Errorf("expected Location %q, got %q", redirects.UnsubscribeError, loc)
		}
	})
}
func TestHandleNewsletterPreview(t *testing.T) {
	t.Run("returns rendered HTML for valid id without token", func(t *testing.T) {
		previewer := &fakeNewsletterPreviewer{body: "<html><body>Hello</body></html>"}
		h := NewPublicHandler(&fakeSubscriber{}, previewer, RedirectPages{}, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/mail/42", nil)
		req.SetPathValue("id", "42")
		w := httptest.NewRecorder()
		h.handleNewsletterPreview(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Errorf("expected text/html content-type, got %q", ct)
		}
		if w.Body.String() != "<html><body>Hello</body></html>" {
			t.Errorf("unexpected body: %s", w.Body)
		}
		if previewer.lastID != 42 {
			t.Errorf("expected id 42, got %d", previewer.lastID)
		}
		if previewer.lastToken != "" {
			t.Errorf("expected empty token, got %q", previewer.lastToken)
		}
	})

	t.Run("passes token to previewer when provided", func(t *testing.T) {
		previewer := &fakeNewsletterPreviewer{body: "<html/>"}
		h := NewPublicHandler(&fakeSubscriber{}, previewer, RedirectPages{}, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/mail/7?token=abc123", nil)
		req.SetPathValue("id", "7")
		w := httptest.NewRecorder()
		h.handleNewsletterPreview(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		if previewer.lastToken != "abc123" {
			t.Errorf("expected token %q, got %q", "abc123", previewer.lastToken)
		}
	})

	t.Run("returns 400 on non-numeric id", func(t *testing.T) {
		h := NewPublicHandler(&fakeSubscriber{}, &fakeNewsletterPreviewer{}, RedirectPages{}, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/mail/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()
		h.handleNewsletterPreview(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 404 when newsletter not found regardless of token", func(t *testing.T) {
		previewer := &fakeNewsletterPreviewer{err: errors.New("not found")}
		h := NewPublicHandler(&fakeSubscriber{}, previewer, RedirectPages{}, slog.Default())

		for _, token := range []string{"", "some-token"} {
			req := httptest.NewRequest(http.MethodGet, "/mail/99?token="+token, nil)
			req.SetPathValue("id", "99")
			w := httptest.NewRecorder()
			h.handleNewsletterPreview(w, req)

			if w.Code != http.StatusNotFound {
				t.Errorf("token=%q: expected 404, got %d", token, w.Code)
			}
		}
	})

	t.Run("preview is routed via Routes()", func(t *testing.T) {
		previewer := &fakeNewsletterPreviewer{body: "<p>hi</p>"}
		h := NewPublicHandler(&fakeSubscriber{}, previewer, RedirectPages{}, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/mail/1", nil)
		w := httptest.NewRecorder()
		h.Routes().ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
	})
}
