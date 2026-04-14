package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/5000K/5000mails/domain"
)

// --- fakes ---

type fakeListManager struct {
	lists map[string]*domain.MailingList
	users []*domain.User

	createErr     error
	getAllErr     error
	getErr        error
	renameErr     error
	deleteErr     error
	countUsersErr error
	usersErr      error
}

func newFakeListManager(lists ...*domain.MailingList) *fakeListManager {
	m := &fakeListManager{lists: make(map[string]*domain.MailingList)}
	for _, l := range lists {
		m.lists[l.Name] = l
	}
	return m
}

func (f *fakeListManager) All(_ context.Context) ([]domain.MailingList, error) {
	if f.getAllErr != nil {
		return nil, f.getAllErr
	}
	out := make([]domain.MailingList, 0, len(f.lists))
	for _, l := range f.lists {
		out = append(out, *l)
	}
	return out, nil
}

func (f *fakeListManager) Create(_ context.Context, name string) (*domain.MailingList, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	l := &domain.MailingList{Name: name}
	f.lists[name] = l
	return l, nil
}

func (f *fakeListManager) Get(_ context.Context, name string) (*domain.MailingList, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	l, ok := f.lists[name]
	if !ok {
		return nil, fmt.Errorf("list %q not found", name)
	}
	return l, nil
}

func (f *fakeListManager) Rename(_ context.Context, name, newName string) (*domain.MailingList, error) {
	if f.renameErr != nil {
		return nil, f.renameErr
	}
	l, ok := f.lists[name]
	if !ok {
		return nil, fmt.Errorf("list %q not found", name)
	}
	delete(f.lists, name)
	l.Name = newName
	f.lists[newName] = l
	return l, nil
}

func (f *fakeListManager) Delete(_ context.Context, name string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.lists[name]; !ok {
		return fmt.Errorf("list %q not found", name)
	}
	delete(f.lists, name)
	return nil
}

func (f *fakeListManager) CountUsers(_ context.Context, listName string) (domain.UserCounts, error) {
	if f.countUsersErr != nil {
		return domain.UserCounts{}, f.countUsersErr
	}
	var total, confirmed int
	for _, u := range f.users {
		if u.MailingListName == listName {
			total++
			if u.IsConfirmed() {
				confirmed++
			}
		}
	}
	return domain.UserCounts{Total: total, Confirmed: confirmed}, nil
}

func (f *fakeListManager) Users(_ context.Context, listName string) ([]domain.User, error) {
	if f.usersErr != nil {
		return nil, f.usersErr
	}
	var out []domain.User
	for _, u := range f.users {
		if u.MailingListName == listName {
			out = append(out, *u)
		}
	}
	return out, nil
}

type fakeMailDispatcher struct {
	sendToListErr   error
	sendTestMailErr error

	lastListName  string
	lastRaw       string
	lastRecipient domain.User
}

func (f *fakeMailDispatcher) SendToList(_ context.Context, listName, raw string, _ map[string]any) error {
	f.lastListName = listName
	f.lastRaw = raw
	return f.sendToListErr
}

func (f *fakeMailDispatcher) SendTestMail(_ context.Context, recipient domain.User, raw string, _ map[string]any) error {
	f.lastRecipient = recipient
	f.lastRaw = raw
	return f.sendTestMailErr
}

type fakeNewsletterArchive struct {
	newsletters map[uint]*domain.SentNewsletter
	nextID      uint

	getAllErr error
	getErr    error
	deleteErr error
}

func newFakeNewsletterArchive(seed ...*domain.SentNewsletter) *fakeNewsletterArchive {
	f := &fakeNewsletterArchive{newsletters: make(map[uint]*domain.SentNewsletter), nextID: 1}
	for _, n := range seed {
		f.newsletters[n.ID] = n
		if n.ID >= f.nextID {
			f.nextID = n.ID + 1
		}
	}
	return f
}

func (f *fakeNewsletterArchive) AllNewsletters(_ context.Context) ([]domain.SentNewsletter, error) {
	if f.getAllErr != nil {
		return nil, f.getAllErr
	}
	out := make([]domain.SentNewsletter, 0, len(f.newsletters))
	for _, n := range f.newsletters {
		out = append(out, *n)
	}
	return out, nil
}

func (f *fakeNewsletterArchive) GetNewsletter(_ context.Context, id uint) (*domain.SentNewsletter, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	n, ok := f.newsletters[id]
	if !ok {
		return nil, fmt.Errorf("newsletter %d not found", id)
	}
	return n, nil
}

func (f *fakeNewsletterArchive) DeleteNewsletter(_ context.Context, id uint) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.newsletters[id]; !ok {
		return fmt.Errorf("newsletter %d not found", id)
	}
	delete(f.newsletters, id)
	return nil
}

// --- helpers ---

type fakeScheduleManager struct{}

func (f *fakeScheduleManager) Schedule(_ context.Context, mailingListName, rawMarkdown string, scheduledAt int64) (*domain.ScheduledMail, error) {
	return &domain.ScheduledMail{ID: 1, MailingListName: mailingListName, RawMarkdown: rawMarkdown, ScheduledAt: scheduledAt}, nil
}
func (f *fakeScheduleManager) List(_ context.Context) ([]domain.ScheduledMail, error) {
	return nil, nil
}
func (f *fakeScheduleManager) Get(_ context.Context, id uint) (*domain.ScheduledMail, error) {
	return &domain.ScheduledMail{ID: id}, nil
}
func (f *fakeScheduleManager) Delete(_ context.Context, _ uint) error { return nil }
func (f *fakeScheduleManager) Reschedule(_ context.Context, id uint, scheduledAt int64) (*domain.ScheduledMail, error) {
	return &domain.ScheduledMail{ID: id, ScheduledAt: scheduledAt}, nil
}
func (f *fakeScheduleManager) ReplaceContent(_ context.Context, id uint, rawMarkdown string) (*domain.ScheduledMail, error) {
	return &domain.ScheduledMail{ID: id, RawMarkdown: rawMarkdown}, nil
}

func newPrivateTestHandler(lists *fakeListManager, mail *fakeMailDispatcher, pub ed25519.PublicKey) *PrivateHandler {
	return NewPrivateHandler(lists, mail, newFakeNewsletterArchive(), &fakeScheduleManager{}, pub, slog.Default())
}

func privateRequest(t *testing.T, h *PrivateHandler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, target, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

func signedPrivateRequest(t *testing.T, h *PrivateHandler, priv ed25519.PrivateKey, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, target, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	ts := fmt.Sprintf("%d", time.Now().Unix())
	msg := buildSignedMessage(ts, method, req.URL.Path, bodyBytes)
	sig := ed25519.Sign(priv, msg)
	req.Header.Set("X-Timestamp", ts)
	req.Header.Set("X-Signature", hex.EncodeToString(sig))

	w := httptest.NewRecorder()
	h.Routes().ServeHTTP(w, req)
	return w
}

func decodeJSON(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.NewDecoder(w.Body).Decode(v); err != nil {
		t.Fatalf("decode response: %v (body: %s)", err, w.Body.String())
	}
}

func TestPrivateHandler_AllLists(t *testing.T) {
	t.Run("returns all lists", func(t *testing.T) {
		m := newFakeListManager(
			&domain.MailingList{Name: "weekly"},
			&domain.MailingList{Name: "monthly"},
		)
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodGet, "/lists", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp []listResponse
		decodeJSON(t, w, &resp)
		if len(resp) != 2 {
			t.Errorf("expected 2 lists, got %d", len(resp))
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		m := newFakeListManager()
		m.getAllErr = errors.New("db failure")
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodGet, "/lists", nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

// --- list tests ---

func TestPrivateHandler_CreateList(t *testing.T) {
	t.Run("returns 201 with created list", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodPost, "/lists", map[string]string{"name": "weekly"})
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
		}
		var resp listResponse
		decodeJSON(t, w, &resp)
		if resp.Name != "weekly" {
			t.Errorf("expected name %q, got %q", "weekly", resp.Name)
		}
	})

	t.Run("returns 400 when name is missing", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodPost, "/lists", map[string]string{})
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		m := newFakeListManager()
		m.createErr = errors.New("db failure")
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodPost, "/lists", map[string]string{"name": "weekly"})
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_GetList(t *testing.T) {
	now := time.Now()
	list := &domain.MailingList{Name: "weekly"}

	t.Run("returns list with stats", func(t *testing.T) {
		m := newFakeListManager(list)
		m.users = []*domain.User{
			{ID: 1, MailingListName: "weekly", Email: "a@test.com", ConfirmedAt: &now},
			{ID: 2, MailingListName: "weekly", Email: "b@test.com"},
		}
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/lists/weekly", nil)
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleGetList(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp listDetailResponse
		decodeJSON(t, w, &resp)
		if resp.Name != "weekly" {
			t.Errorf("unexpected list: %+v", resp)
		}
		if resp.Subscribers.Total != 2 || resp.Subscribers.Confirmed != 1 {
			t.Errorf("unexpected counts: total=%d confirmed=%d", resp.Subscribers.Total, resp.Subscribers.Confirmed)
		}
	})

	t.Run("returns 404 when list not found", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/lists/ghost", nil)
		req.SetPathValue("name", "ghost")
		w := httptest.NewRecorder()
		h.handleGetList(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_RenameList(t *testing.T) {
	t.Run("returns updated list", func(t *testing.T) {
		m := newFakeListManager(&domain.MailingList{Name: "old"})
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodPut, "/lists/old", jsonBody(t, map[string]string{"name": "new"}))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("name", "old")
		w := httptest.NewRecorder()
		h.handleRenameList(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp listResponse
		decodeJSON(t, w, &resp)
		if resp.Name != "new" {
			t.Errorf("expected name %q, got %q", "new", resp.Name)
		}
	})

	t.Run("returns 400 when name missing", func(t *testing.T) {
		m := newFakeListManager(&domain.MailingList{Name: "old"})
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodPut, "/lists/old", jsonBody(t, map[string]string{}))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("name", "old")
		w := httptest.NewRecorder()
		h.handleRenameList(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_DeleteList(t *testing.T) {
	t.Run("returns 204", func(t *testing.T) {
		m := newFakeListManager(&domain.MailingList{Name: "bye"})
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodDelete, "/lists/bye", nil)
		req.SetPathValue("name", "bye")
		w := httptest.NewRecorder()
		h.handleDeleteList(w, req)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
		if _, exists := m.lists["bye"]; exists {
			t.Error("list should have been deleted")
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		m := newFakeListManager(&domain.MailingList{Name: "bye"})
		m.deleteErr = errors.New("db down")
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodDelete, "/lists/bye", nil)
		req.SetPathValue("name", "bye")
		w := httptest.NewRecorder()
		h.handleDeleteList(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_ListUsers(t *testing.T) {
	now := time.Now()
	list := &domain.MailingList{Name: "weekly"}

	t.Run("returns all users with confirmed flag", func(t *testing.T) {
		m := newFakeListManager(list)
		m.users = []*domain.User{
			{ID: 1, MailingListName: "weekly", Name: "Alice", Email: "a@test.com", ConfirmedAt: &now},
			{ID: 2, MailingListName: "weekly", Name: "Bob", Email: "b@test.com"},
		}
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/lists/weekly/users", nil)
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleListUsers(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp []userResponse
		decodeJSON(t, w, &resp)
		if len(resp) != 2 {
			t.Fatalf("expected 2 users, got %d", len(resp))
		}
		if !resp[0].Confirmed {
			t.Error("expected first user to be confirmed")
		}
		if resp[1].Confirmed {
			t.Error("expected second user to be unconfirmed")
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		m := newFakeListManager(list)
		m.usersErr = errors.New("db down")
		h := newPrivateTestHandler(m, &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodGet, "/lists/weekly/users", nil)
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleListUsers(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_SendToList(t *testing.T) {
	t.Run("dispatches mail and returns 200", func(t *testing.T) {
		mail := &fakeMailDispatcher{}
		h := newPrivateTestHandler(newFakeListManager(), mail, nil)
		req := httptest.NewRequest(http.MethodPost, "/lists/weekly/send", jsonBody(t, map[string]any{"raw": "# Hello"}))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleSendToList(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		if mail.lastListName != "weekly" {
			t.Errorf("expected list %q, got %q", "weekly", mail.lastListName)
		}
		if mail.lastRaw != "# Hello" {
			t.Errorf("unexpected raw: %q", mail.lastRaw)
		}
	})

	t.Run("returns 400 when raw is missing", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/lists/weekly/send", jsonBody(t, map[string]any{}))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleSendToList(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mail := &fakeMailDispatcher{sendToListErr: errors.New("smtp down")}
		h := newPrivateTestHandler(newFakeListManager(), mail, nil)
		req := httptest.NewRequest(http.MethodPost, "/lists/weekly/send", jsonBody(t, map[string]any{"raw": "# Hello"}))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("name", "weekly")
		w := httptest.NewRecorder()
		h.handleSendToList(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_SendTestMail(t *testing.T) {
	t.Run("sends test mail and returns 200", func(t *testing.T) {
		mail := &fakeMailDispatcher{}
		h := newPrivateTestHandler(newFakeListManager(), mail, nil)
		req := httptest.NewRequest(http.MethodPost, "/mail/test", jsonBody(t, map[string]any{
			"recipient": map[string]string{"name": "Alice", "email": "alice@test.com"},
			"raw":       "# Hi",
		}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.handleSendTestMail(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		if mail.lastRecipient.Email != "alice@test.com" {
			t.Errorf("expected email %q, got %q", "alice@test.com", mail.lastRecipient.Email)
		}
	})

	t.Run("returns 400 when recipient email is missing", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		req := httptest.NewRequest(http.MethodPost, "/mail/test", jsonBody(t, map[string]any{"raw": "# Hi"}))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.handleSendTestMail(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// --- newsletter tests ---

func TestPrivateHandler_AllNewsletters(t *testing.T) {
	now := time.Now()
	n := &domain.SentNewsletter{
		ID:           1,
		Subject:      "Weekly Digest",
		SenderName:   "Bot",
		SentAt:       now,
		MailingLists: []domain.MailingList{{Name: "weekly"}},
	}

	t.Run("returns all newsletters", func(t *testing.T) {
		nl := newFakeNewsletterArchive(n)
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, nl, &fakeScheduleManager{}, nil, slog.Default())
		w := privateRequest(t, h, http.MethodGet, "/newsletters", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp []newsletterSummaryResponse
		decodeJSON(t, w, &resp)
		if len(resp) != 1 {
			t.Fatalf("expected 1 newsletter, got %d", len(resp))
		}
		if resp[0].Subject != "Weekly Digest" {
			t.Errorf("expected subject %q, got %q", "Weekly Digest", resp[0].Subject)
		}
		if len(resp[0].MailingLists) != 1 || resp[0].MailingLists[0] != "weekly" {
			t.Errorf("unexpected mailing lists: %v", resp[0].MailingLists)
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		nl := newFakeNewsletterArchive()
		nl.getAllErr = errors.New("db failure")
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, nl, &fakeScheduleManager{}, nil, slog.Default())
		w := privateRequest(t, h, http.MethodGet, "/newsletters", nil)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_GetNewsletter(t *testing.T) {
	now := time.Now()
	confirmed := now
	n := &domain.SentNewsletter{
		ID:           42,
		Subject:      "Issue #1",
		SenderName:   "Editor",
		RawMarkdown:  "# Hello",
		SentAt:       now,
		Recipients:   []domain.User{{ID: 1, Name: "Alice", Email: "alice@example.com", ConfirmedAt: &confirmed}},
		MailingLists: []domain.MailingList{{Name: "monthly"}},
	}

	t.Run("returns newsletter detail", func(t *testing.T) {
		nl := newFakeNewsletterArchive(n)
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, nl, &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/newsletters/42", nil)
		req.SetPathValue("id", "42")
		w := httptest.NewRecorder()
		h.handleGetNewsletter(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
		}
		var resp newsletterDetailResponse
		decodeJSON(t, w, &resp)
		if resp.Subject != "Issue #1" {
			t.Errorf("expected subject %q, got %q", "Issue #1", resp.Subject)
		}
		if resp.RawMarkdown != "# Hello" {
			t.Errorf("expected raw %q, got %q", "# Hello", resp.RawMarkdown)
		}
		if len(resp.Recipients) != 1 || resp.Recipients[0].Email != "alice@example.com" {
			t.Errorf("unexpected recipients: %+v", resp.Recipients)
		}
		if len(resp.MailingLists) != 1 || resp.MailingLists[0] != "monthly" {
			t.Errorf("unexpected mailing lists: %v", resp.MailingLists)
		}
	})

	t.Run("returns 404 when not found", func(t *testing.T) {
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, newFakeNewsletterArchive(), &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/newsletters/99", nil)
		req.SetPathValue("id", "99")
		w := httptest.NewRecorder()
		h.handleGetNewsletter(w, req)
		if w.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, newFakeNewsletterArchive(), &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodGet, "/newsletters/abc", nil)
		req.SetPathValue("id", "abc")
		w := httptest.NewRecorder()
		h.handleGetNewsletter(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestPrivateHandler_DeleteNewsletter(t *testing.T) {
	now := time.Now()
	n := &domain.SentNewsletter{ID: 7, Subject: "Old", SentAt: now}

	t.Run("returns 204 on success", func(t *testing.T) {
		nl := newFakeNewsletterArchive(n)
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, nl, &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodDelete, "/newsletters/7", nil)
		req.SetPathValue("id", "7")
		w := httptest.NewRecorder()
		h.handleDeleteNewsletter(w, req)
		if w.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", w.Code)
		}
		if _, exists := nl.newsletters[7]; exists {
			t.Error("newsletter should have been deleted")
		}
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		nl := newFakeNewsletterArchive(n)
		nl.deleteErr = errors.New("db down")
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, nl, &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodDelete, "/newsletters/7", nil)
		req.SetPathValue("id", "7")
		w := httptest.NewRecorder()
		h.handleDeleteNewsletter(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("returns 400 on invalid id", func(t *testing.T) {
		h := NewPrivateHandler(newFakeListManager(), &fakeMailDispatcher{}, newFakeNewsletterArchive(), &fakeScheduleManager{}, nil, slog.Default())
		req := httptest.NewRequest(http.MethodDelete, "/newsletters/bad", nil)
		req.SetPathValue("id", "bad")
		w := httptest.NewRecorder()
		h.handleDeleteNewsletter(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

// --- authentication tests ---

func TestPrivateHandler_Auth(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("rejects unsigned request when key configured", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, pub)
		w := privateRequest(t, h, http.MethodPost, "/lists", map[string]string{"name": "test"})
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("accepts correctly signed request", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, pub)
		w := signedPrivateRequest(t, h, priv, http.MethodPost, "/lists", map[string]string{"name": "signed"})
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body)
		}
	})

	t.Run("rejects tampered signature", func(t *testing.T) {
		_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, pub)
		w := signedPrivateRequest(t, h, otherPriv, http.MethodPost, "/lists", map[string]string{"name": "evil"})
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("allows requests without key configured", func(t *testing.T) {
		h := newPrivateTestHandler(newFakeListManager(), &fakeMailDispatcher{}, nil)
		w := privateRequest(t, h, http.MethodPost, "/lists", map[string]string{"name": "open"})
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body)
		}
	})
}

// --- client integration test ---

func TestPrivateClient_Integration(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	m := newFakeListManager(&domain.MailingList{Name: "weekly"})
	m.users = []*domain.User{
		{ID: 1, MailingListName: "weekly", Name: "Alice", Email: "a@test.com", ConfirmedAt: &now},
	}
	mail := &fakeMailDispatcher{}

	srv := httptest.NewServer(NewPrivateHandler(m, mail, newFakeNewsletterArchive(), &fakeScheduleManager{}, pub, slog.Default()).Routes())
	defer srv.Close()

	client := NewPrivateClient(srv.URL, priv)
	ctx := context.Background()

	t.Run("CreateList", func(t *testing.T) {
		resp, err := client.CreateList(ctx, "monthly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "monthly" {
			t.Errorf("expected %q, got %q", "monthly", resp.Name)
		}
	})

	t.Run("GetList", func(t *testing.T) {
		resp, err := client.GetList(ctx, "weekly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "weekly" {
			t.Errorf("expected %q, got %q", "weekly", resp.Name)
		}
		if resp.Subscribers.Total != 1 || resp.Subscribers.Confirmed != 1 {
			t.Errorf("unexpected counts: %+v", resp.Subscribers)
		}
	})

	t.Run("GetUsers", func(t *testing.T) {
		users, err := client.GetUsers(ctx, "weekly")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(users) != 1 || users[0].Email != "a@test.com" {
			t.Errorf("unexpected users: %+v", users)
		}
	})

	t.Run("SendToList", func(t *testing.T) {
		if err := client.SendToList(ctx, "weekly", "# Hello", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mail.lastListName != "weekly" {
			t.Errorf("expected list %q, got %q", "weekly", mail.lastListName)
		}
	})

	t.Run("SendTestMail", func(t *testing.T) {
		if err := client.SendTestMail(ctx, RecipientInput{Name: "Bob", Email: "bob@test.com"}, "# Test", nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mail.lastRecipient.Email != "bob@test.com" {
			t.Errorf("expected email %q, got %q", "bob@test.com", mail.lastRecipient.Email)
		}
	})

	t.Run("RenameList", func(t *testing.T) {
		resp, err := client.RenameList(ctx, "weekly", "renamed")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "renamed" {
			t.Errorf("expected %q, got %q", "renamed", resp.Name)
		}
	})

	t.Run("DeleteList", func(t *testing.T) {
		_, _ = client.CreateList(ctx, "todelete")
		if err := client.DeleteList(ctx, "todelete"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := m.lists["todelete"]; exists {
			t.Error("list should have been deleted")
		}
	})

	t.Run("unauthenticated client is rejected", func(t *testing.T) {
		unauthClient := NewPrivateClient(srv.URL, nil)
		_, err := unauthClient.CreateList(ctx, "nope")
		if err == nil {
			t.Error("expected error for unsigned request")
		}
	})
}

// --- helper ---

func jsonBody(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		t.Fatal(err)
	}
	return &buf
}
