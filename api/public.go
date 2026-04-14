package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/5000K/5000mails/domain"
)

type Subscriber interface {
	Subscribe(ctx context.Context, listName, userName, email string) (*domain.User, error)
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, unsubscribeToken string) error
}

type RedirectPages struct {
	SubscribeSuccess   string
	SubscribeError     string
	ConfirmSuccess     string
	ConfirmError       string
	UnsubscribeSuccess string
	UnsubscribeError   string
}

type PublicHandler struct {
	subscriptions Subscriber
	redirects     RedirectPages
	logger        *slog.Logger
}

func NewPublicHandler(subscriptions Subscriber, redirects RedirectPages, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{subscriptions: subscriptions, redirects: redirects, logger: logger}
}

func (h *PublicHandler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{listName}/subscribe", h.handleSubscribe)
	mux.HandleFunc("GET /confirm/{token}", h.handleConfirm)
	mux.HandleFunc("GET /unsubscribe/{token}", h.handleUnsubscribe)
	return mux
}

func (h *PublicHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	listName := r.PathValue("listName")

	name, email, err := parseSubscribeBody(r)
	if err != nil {
		redirectOrError(w, r, h.redirects.SubscribeError, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.subscriptions.Subscribe(r.Context(), listName, name, email); err != nil {
		h.logger.ErrorContext(r.Context(), "subscribe failed",
			slog.String("list", listName),
			slog.String("email", email),
			slog.Any("error", err),
		)
		redirectOrError(w, r, h.redirects.SubscribeError, http.StatusInternalServerError, "subscription failed")
		return
	}

	redirectOrJSON(w, r, h.redirects.SubscribeSuccess, http.StatusAccepted, map[string]string{"message": "check your email for a confirmation link"})
}

func (h *PublicHandler) handleConfirm(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	if err := h.subscriptions.Confirm(r.Context(), token); err != nil {
		h.logger.ErrorContext(r.Context(), "confirm failed",
			slog.String("token", token),
			slog.Any("error", err),
		)
		redirectOrError(w, r, h.redirects.ConfirmError, http.StatusBadRequest, "invalid or expired confirmation token")
		return
	}

	redirectOrJSON(w, r, h.redirects.ConfirmSuccess, http.StatusOK, map[string]string{"message": "your subscription has been confirmed"})
}

func (h *PublicHandler) handleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	if err := h.subscriptions.Unsubscribe(r.Context(), token); err != nil {
		h.logger.ErrorContext(r.Context(), "unsubscribe failed",
			slog.String("token", token),
			slog.Any("error", err),
		)
		redirectOrError(w, r, h.redirects.UnsubscribeError, http.StatusBadRequest, "invalid or expired unsubscribe token")
		return
	}

	redirectOrJSON(w, r, h.redirects.UnsubscribeSuccess, http.StatusOK, map[string]string{"message": "you have been unsubscribed"})
}

func redirectOrJSON(w http.ResponseWriter, r *http.Request, redirectURL string, status int, v any) {
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	writeJSON(w, status, v)
}

func redirectOrError(w http.ResponseWriter, r *http.Request, redirectURL string, status int, msg string) {
	if redirectURL != "" {
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return
	}
	writeError(w, status, msg)
}

func parseSubscribeBody(r *http.Request) (name, email string, err error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var body struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return "", "", fmt.Errorf("invalid request body")
		}
		name, email = body.Name, body.Email
	} else {
		if err := r.ParseForm(); err != nil {
			return "", "", fmt.Errorf("invalid form data")
		}
		name, email = r.FormValue("name"), r.FormValue("email")
	}
	if name == "" || email == "" {
		return "", "", fmt.Errorf("name and email are required")
	}
	return name, email, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
