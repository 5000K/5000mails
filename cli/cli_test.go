package cli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/5000K/5000mails/api"
	"github.com/5000K/5000mails/domain"
)

// --- fakes (same shape as api tests) ---

type fakeListManager struct {
	lists  map[uint]*domain.MailingList
	nextID uint
	users  []*domain.User
}

func newFakeListManager(lists ...*domain.MailingList) *fakeListManager {
	m := &fakeListManager{lists: make(map[uint]*domain.MailingList), nextID: 1}
	for _, l := range lists {
		m.lists[l.ID] = l
		if l.ID >= m.nextID {
			m.nextID = l.ID + 1
		}
	}
	return m
}

func (f *fakeListManager) Create(_ context.Context, name string) (*domain.MailingList, error) {
	l := &domain.MailingList{ID: f.nextID, Name: name}
	f.nextID++
	f.lists[l.ID] = l
	return l, nil
}

func (f *fakeListManager) Get(_ context.Context, id uint) (*domain.MailingList, error) {
	l, ok := f.lists[id]
	if !ok {
		return nil, fmt.Errorf("list %d not found", id)
	}
	return l, nil
}

func (f *fakeListManager) Rename(_ context.Context, id uint, name string) (*domain.MailingList, error) {
	l, ok := f.lists[id]
	if !ok {
		return nil, fmt.Errorf("list %d not found", id)
	}
	l.Name = name
	return l, nil
}

func (f *fakeListManager) Delete(_ context.Context, id uint) error {
	if _, ok := f.lists[id]; !ok {
		return fmt.Errorf("list %d not found", id)
	}
	delete(f.lists, id)
	return nil
}

func (f *fakeListManager) CountUsers(_ context.Context, listID uint) (domain.UserCounts, error) {
	var total, confirmed int
	for _, u := range f.users {
		if u.MailingListID == listID {
			total++
			if u.IsConfirmed() {
				confirmed++
			}
		}
	}
	return domain.UserCounts{Total: total, Confirmed: confirmed}, nil
}

func (f *fakeListManager) Users(_ context.Context, listID uint) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.users {
		if u.MailingListID == listID {
			out = append(out, *u)
		}
	}
	return out, nil
}

type fakeMailDispatcher struct {
	lastListName  string
	lastRaw       string
	lastRecipient domain.User
}

func (f *fakeMailDispatcher) SendToList(_ context.Context, listName, raw string, _ map[string]any) error {
	f.lastListName = listName
	f.lastRaw = raw
	return nil
}

func (f *fakeMailDispatcher) SendTestMail(_ context.Context, r domain.User, raw string, _ map[string]any) error {
	f.lastRecipient = r
	f.lastRaw = raw
	return nil
}

// --- helpers ---

func startTestServer(t *testing.T, lm *fakeListManager, md *fakeMailDispatcher, pub ed25519.PublicKey) *httptest.Server {
	t.Helper()
	h := api.NewPrivateHandler(lm, md, pub, slog.Default())
	return httptest.NewServer(h.Routes())
}

func tmpRawFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "mail.md")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// --- key generation tests ---

func TestGenerateKeyPair(t *testing.T) {
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("unexpected public key size: %d", len(pub))
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("unexpected private key size: %d", len(priv))
	}
}

func TestWriteAndReadKeyPair(t *testing.T) {
	dir := t.TempDir()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)

	pubPath, privPath, err := WriteKeyPair(dir, pub, priv)
	if err != nil {
		t.Fatal(err)
	}

	readPriv, err := ReadPrivateKey(privPath)
	if err != nil {
		t.Fatalf("read private key: %v", err)
	}
	if !readPriv.Equal(priv) {
		t.Error("private key round-trip mismatch")
	}

	readPub, err := ReadPublicKey(pubPath)
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	if !readPub.Equal(pub) {
		t.Error("public key round-trip mismatch")
	}
}

func TestKeysGenerateSubcommand(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	code := Run([]string{"keys", "generate", "--out-dir", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "private key:") {
		t.Errorf("expected private key path in output, got: %s", stdout.String())
	}

	if _, err := os.Stat(filepath.Join(dir, privateKeyFile)); err != nil {
		t.Errorf("private key file not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, publicKeyFile)); err != nil {
		t.Errorf("public key file not created: %v", err)
	}
}

// --- cli dispatch tests ---

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage: 5kmcli") {
		t.Error("expected usage in output")
	}
}

func TestRunNoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run(nil, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bogus"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("expected 'unknown command' in stderr, got: %s", stderr.String())
	}
}

// --- list subcommand integration ---

func TestListCreate(t *testing.T) {
	m := newFakeListManager()
	srv := startTestServer(t, m, &fakeMailDispatcher{}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "list", "create", "--name", "weekly"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	var resp api.ListResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Name != "weekly" {
		t.Errorf("expected %q, got %q", "weekly", resp.Name)
	}
}

func TestListGet(t *testing.T) {
	now := time.Now()
	m := newFakeListManager(&domain.MailingList{ID: 1, Name: "weekly"})
	m.users = []*domain.User{
		{ID: 1, MailingListID: 1, Email: "a@test.com", ConfirmedAt: &now},
	}
	srv := startTestServer(t, m, &fakeMailDispatcher{}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "list", "get", "--id", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	var resp api.ListDetailResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Subscribers.Total != 1 || resp.Subscribers.Confirmed != 1 {
		t.Errorf("unexpected subscriber stats: %+v", resp.Subscribers)
	}
}

func TestListRename(t *testing.T) {
	m := newFakeListManager(&domain.MailingList{ID: 1, Name: "old"})
	srv := startTestServer(t, m, &fakeMailDispatcher{}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "list", "rename", "--id", "1", "--name", "new"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	var resp api.ListResponse
	json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Name != "new" {
		t.Errorf("expected %q, got %q", "new", resp.Name)
	}
}

func TestListDelete(t *testing.T) {
	m := newFakeListManager(&domain.MailingList{ID: 1, Name: "bye"})
	srv := startTestServer(t, m, &fakeMailDispatcher{}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "list", "delete", "--id", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	if _, exists := m.lists[1]; exists {
		t.Error("list should have been deleted")
	}
}

func TestListUsers(t *testing.T) {
	now := time.Now()
	m := newFakeListManager(&domain.MailingList{ID: 1, Name: "weekly"})
	m.users = []*domain.User{
		{ID: 1, MailingListID: 1, Name: "Alice", Email: "a@test.com", ConfirmedAt: &now},
		{ID: 2, MailingListID: 1, Name: "Bob", Email: "b@test.com"},
	}
	srv := startTestServer(t, m, &fakeMailDispatcher{}, nil)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "list", "users", "--id", "1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	var users []api.UserItem
	json.Unmarshal(stdout.Bytes(), &users)
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

// --- send subcommands ---

func TestSendToList(t *testing.T) {
	mail := &fakeMailDispatcher{}
	srv := startTestServer(t, newFakeListManager(), mail, nil)
	defer srv.Close()

	rawFile := tmpRawFile(t, "# Hello")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "send", "list", "--list", "weekly", "--raw-path", rawFile, "--data", "foo=bar"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	if mail.lastListName != "weekly" {
		t.Errorf("expected list %q, got %q", "weekly", mail.lastListName)
	}
	if mail.lastRaw != "# Hello" {
		t.Errorf("expected raw %q, got %q", "# Hello", mail.lastRaw)
	}
	if !strings.Contains(stdout.String(), "mail dispatched") {
		t.Errorf("expected 'mail dispatched' in output, got: %s", stdout.String())
	}
}

func TestSendTestMail(t *testing.T) {
	mail := &fakeMailDispatcher{}
	srv := startTestServer(t, newFakeListManager(), mail, nil)
	defer srv.Close()

	rawFile := tmpRawFile(t, "# Test")
	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"--server", srv.URL,
		"send", "test",
		"--name", "Alice",
		"--email", "alice@test.com",
		"--raw-path", rawFile,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	if mail.lastRecipient.Email != "alice@test.com" {
		t.Errorf("expected email %q, got %q", "alice@test.com", mail.lastRecipient.Email)
	}
	if mail.lastRecipient.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", mail.lastRecipient.Name)
	}
}

// --- authenticated CLI ---

func TestAuthenticatedCLI(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	dir := t.TempDir()
	_, privPath, _ := WriteKeyPair(dir, pub, priv)

	m := newFakeListManager()
	srv := startTestServer(t, m, &fakeMailDispatcher{}, pub)
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Run([]string{
		"--server", srv.URL,
		"--private-key-path", privPath,
		"list", "create", "--name", "secure",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, stderr.String())
	}
	var resp api.ListResponse
	json.Unmarshal(stdout.Bytes(), &resp)
	if resp.Name != "secure" {
		t.Errorf("expected %q, got %q", "secure", resp.Name)
	}

	t.Run("rejected without key", func(t *testing.T) {
		var stdout2, stderr2 bytes.Buffer
		code := Run([]string{"--server", srv.URL, "list", "create", "--name", "nope"}, &stdout2, &stderr2)
		if code != 1 {
			t.Errorf("expected 1, got %d", code)
		}
	})
}

// --- flag parser tests ---

func TestParseGlobalFlags(t *testing.T) {
	t.Run("extracts server and key path", func(t *testing.T) {
		var server, key string
		rest := parseGlobalFlags([]string{"--server", "http://x", "--private-key-path", "/k", "list", "create"}, &server, &key)
		if server != "http://x" {
			t.Errorf("server = %q", server)
		}
		if key != "/k" {
			t.Errorf("key = %q", key)
		}
		if len(rest) != 2 || rest[0] != "list" {
			t.Errorf("rest = %v", rest)
		}
	})

	t.Run("returns all args when no global flags", func(t *testing.T) {
		var s, k string
		rest := parseGlobalFlags([]string{"list", "create"}, &s, &k)
		if len(rest) != 2 {
			t.Errorf("rest = %v", rest)
		}
	})
}

func TestCollectData(t *testing.T) {
	args := []string{"--data", "a=1", "--data", "b=hello world", "--other", "x"}
	data := collectData(args)
	if data["a"] != "1" || data["b"] != "hello world" {
		t.Errorf("unexpected data: %v", data)
	}
}

func TestFlagValue(t *testing.T) {
	if v := flagValue([]string{"--name", "test", "--id", "5"}, "--name"); v != "test" {
		t.Errorf("expected %q, got %q", "test", v)
	}
	if v := flagValue([]string{"--name"}, "--name"); v != "" {
		t.Errorf("expected empty, got %q", v)
	}
}
