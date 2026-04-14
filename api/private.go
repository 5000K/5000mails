package api

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/5000K/5000mails/domain"
)

// ListManager is the private API's view of the list service.
type ListManager interface {
	Create(ctx context.Context, name string) (*domain.MailingList, error)
	Get(ctx context.Context, name string) (*domain.MailingList, error)
	Rename(ctx context.Context, name, newName string) (*domain.MailingList, error)
	Delete(ctx context.Context, name string) error
	CountUsers(ctx context.Context, listName string) (domain.UserCounts, error)
	Users(ctx context.Context, listName string) ([]domain.User, error)
}

// MailDispatcher is the private API's view of the mail service.
type MailDispatcher interface {
	SendToList(ctx context.Context, listName string, raw string, data map[string]any) error
	SendTestMail(ctx context.Context, recipient domain.User, raw string, data map[string]any) error
}

// PrivateHandler serves the private admin API.
// When publicKey is non-nil, every request must carry a valid Ed25519 signature.
type PrivateHandler struct {
	lists     ListManager
	mail      MailDispatcher
	publicKey ed25519.PublicKey
	logger    *slog.Logger
}

// NewPrivateHandler creates a new PrivateHandler.
// Pass a nil publicKey to disable request authentication.
func NewPrivateHandler(lists ListManager, mail MailDispatcher, publicKey ed25519.PublicKey, logger *slog.Logger) *PrivateHandler {
	return &PrivateHandler{lists: lists, mail: mail, publicKey: publicKey, logger: logger}
}

// Routes returns the mux for all private API endpoints.
func (h *PrivateHandler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("POST /lists", h.auth(h.handleCreateList))
	mux.Handle("GET /lists/{name}", h.auth(h.handleGetList))
	mux.Handle("PUT /lists/{name}", h.auth(h.handleRenameList))
	mux.Handle("DELETE /lists/{name}", h.auth(h.handleDeleteList))
	mux.Handle("GET /lists/{name}/users", h.auth(h.handleListUsers))
	mux.Handle("POST /lists/{name}/send", h.auth(h.handleSendToList))
	mux.Handle("POST /mail/test", h.auth(h.handleSendTestMail))
	return mux
}

// --- request/response types ---

type listResponse struct {
	Name string `json:"name"`
}

type listDetailResponse struct {
	Name        string `json:"name"`
	Subscribers struct {
		Total     int `json:"total"`
		Confirmed int `json:"confirmed"`
	} `json:"subscribers"`
}

type userResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Confirmed bool   `json:"confirmed"`
}

type sendRequest struct {
	Raw  string         `json:"raw"`
	Data map[string]any `json:"data"`
}

type testMailRequest struct {
	Recipient struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"recipient"`
	Raw  string         `json:"raw"`
	Data map[string]any `json:"data"`
}

// --- handlers ---

func (h *PrivateHandler) handleCreateList(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := h.lists.Create(r.Context(), body.Name)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "create list failed", slog.String("name", body.Name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to create list")
		return
	}

	writeJSON(w, http.StatusCreated, listResponse{Name: list.Name})
}

func (h *PrivateHandler) handleGetList(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	list, err := h.lists.Get(r.Context(), name)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "get list failed", slog.String("name", name), slog.Any("error", err))
		writeError(w, http.StatusNotFound, "list not found")
		return
	}

	counts, err := h.lists.CountUsers(r.Context(), name)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "count users failed", slog.String("name", name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to load list stats")
		return
	}

	resp := listDetailResponse{Name: list.Name}
	resp.Subscribers.Total = counts.Total
	resp.Subscribers.Confirmed = counts.Confirmed
	writeJSON(w, http.StatusOK, resp)
}

func (h *PrivateHandler) handleRenameList(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	list, err := h.lists.Rename(r.Context(), name, body.Name)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "rename list failed", slog.String("name", name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to rename list")
		return
	}

	writeJSON(w, http.StatusOK, listResponse{Name: list.Name})
}

func (h *PrivateHandler) handleDeleteList(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	if err := h.lists.Delete(r.Context(), name); err != nil {
		h.logger.ErrorContext(r.Context(), "delete list failed", slog.String("name", name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to delete list")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *PrivateHandler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	users, err := h.lists.Users(r.Context(), name)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "list users failed", slog.String("name", name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	resp := make([]userResponse, len(users))
	for i, u := range users {
		resp[i] = userResponse{ID: u.ID, Name: u.Name, Email: u.Email, Confirmed: u.IsConfirmed()}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *PrivateHandler) handleSendToList(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var body sendRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Raw == "" {
		writeError(w, http.StatusBadRequest, "raw is required")
		return
	}

	if err := h.mail.SendToList(r.Context(), name, body.Raw, body.Data); err != nil {
		h.logger.ErrorContext(r.Context(), "send to list failed", slog.String("list", name), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to send mail")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "mail dispatched"})
}

func (h *PrivateHandler) handleSendTestMail(w http.ResponseWriter, r *http.Request) {
	var body testMailRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Recipient.Email == "" || body.Raw == "" {
		writeError(w, http.StatusBadRequest, "recipient.email and raw are required")
		return
	}

	recipient := domain.User{Name: body.Recipient.Name, Email: body.Recipient.Email}
	if err := h.mail.SendTestMail(r.Context(), recipient, body.Raw, body.Data); err != nil {
		h.logger.ErrorContext(r.Context(), "send test mail failed", slog.String("email", body.Recipient.Email), slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to send test mail")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "test mail sent"})
}

// --- auth middleware ---

const signatureWindow = 5 * time.Minute

// auth wraps a handler with Ed25519 signature verification when a public key
// is configured. Without a public key the handler is passed through unchanged.
func (h *PrivateHandler) auth(next http.HandlerFunc) http.Handler {
	if len(h.publicKey) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := h.verifySignature(r); err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		next(w, r)
	})
}

func (h *PrivateHandler) verifySignature(r *http.Request) error {
	tsStr := r.Header.Get("X-Timestamp")
	sigHex := r.Header.Get("X-Signature")
	if tsStr == "" || sigHex == "" {
		return fmt.Errorf("missing X-Timestamp or X-Signature header")
	}

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid X-Timestamp")
	}
	age := time.Since(time.Unix(ts, 0))
	if age < -signatureWindow || age > signatureWindow {
		return fmt.Errorf("request timestamp out of acceptable window")
	}

	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return fmt.Errorf("invalid X-Signature encoding")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	msg := buildSignedMessage(tsStr, r.Method, r.URL.Path, body)
	if !ed25519.Verify(h.publicKey, msg, sig) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func buildSignedMessage(timestamp, method, path string, body []byte) []byte {
	sum := sha256.Sum256(body)
	bodyHash := hex.EncodeToString(sum[:])
	return []byte(timestamp + "\n" + method + "\n" + path + "\n" + bodyHash)
}
