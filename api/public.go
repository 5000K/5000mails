package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/5000K/5000mails/domain"
)

type Subscriber interface {
	Subscribe(ctx context.Context, listName, userName, email string, topicNames []string) (*domain.User, error)
	Confirm(ctx context.Context, token string) error
	Unsubscribe(ctx context.Context, unsubscribeToken string) error
}

type NewsletterPreviewer interface {
	RenderNewsletter(ctx context.Context, id uint, unsubscribeToken string) (string, error)
}

type PreferencesManager interface {
	GetUserTopics(ctx context.Context, mailingListName string, userID uint) ([]domain.Topic, error)
	SetUserTopics(ctx context.Context, mailingListName string, userID uint, topicIDs []uint) error
	List(ctx context.Context, mailingListName string) ([]domain.Topic, error)
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
	newsletters   NewsletterPreviewer
	preferences   PreferencesManager
	users         domain.UserRepository
	renderer      domain.Renderer
	redirects     RedirectPages
	logger        *slog.Logger
}

func NewPublicHandler(subscriptions Subscriber, newsletters NewsletterPreviewer, preferences PreferencesManager, users domain.UserRepository, renderer domain.Renderer, redirects RedirectPages, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{subscriptions: subscriptions, newsletters: newsletters, preferences: preferences, users: users, renderer: renderer, redirects: redirects, logger: logger}
}

func (h *PublicHandler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /{listName}/subscribe", h.handleSubscribe)
	mux.HandleFunc("GET /confirm/{token}", h.handleConfirm)
	mux.HandleFunc("GET /unsubscribe/{token}", h.handleUnsubscribe)
	mux.HandleFunc("GET /mail/{id}", h.handleNewsletterPreview)
	mux.HandleFunc("GET /preferences/{listName}/{token}", h.handlePreferencesPage)
	mux.HandleFunc("POST /preferences/{listName}/{token}", h.handleSavePreferences)
	return mux
}

func (h *PublicHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	listName := r.PathValue("listName")

	name, email, err := parseSubscribeBody(r)
	if err != nil {
		redirectOrError(w, r, h.redirects.SubscribeError, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := h.subscriptions.Subscribe(r.Context(), listName, name, email, nil); err != nil {
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

func (h *PublicHandler) handleNewsletterPreview(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid newsletter id")
		return
	}

	token := r.URL.Query().Get("token")
	body, err := h.newsletters.RenderNewsletter(r.Context(), uint(id), token)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "newsletter preview failed",
			slog.Uint64("id", id),
			slog.Any("error", err),
		)
		writeError(w, http.StatusNotFound, "newsletter not found")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, body)
}

var preferencesTemplate = `<h1>Topic Preferences</h1>
<form method="POST">
{{range .topics}}
<label><input type="checkbox" name="topic" value="{{.ID}}"{{if .Subscribed}} checked{{end}}> {{.DisplayName}}</label><br>
{{end}}
<button type="submit">Save</button>
</form>
{{if .saved}}<p>Preferences saved.</p>{{end}}`

type preferencesTopicData struct {
	ID          uint
	Name        string
	DisplayName string
	Subscribed  bool
}

func (h *PublicHandler) handlePreferencesPage(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	saved := r.URL.Query().Get("saved") == "1"

	user, err := h.users.GetUserByUnsubscribeToken(r.Context(), token)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "preferences: user lookup failed", slog.Any("error", err))
		writeError(w, http.StatusNotFound, "invalid token")
		return
	}

	allTopics, err := h.preferences.List(r.Context(), user.MailingListName)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "preferences: list topics failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to load topics")
		return
	}

	userTopics, err := h.preferences.GetUserTopics(r.Context(), user.MailingListName, user.ID)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "preferences: get user topics failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to load preferences")
		return
	}

	subscribedIDs := make(map[uint]bool)
	for _, t := range userTopics {
		subscribedIDs[t.ID] = true
	}

	topicData := make([]preferencesTopicData, len(allTopics))
	for i, t := range allTopics {
		topicData[i] = preferencesTopicData{
			ID:          t.ID,
			Name:        t.Name,
			DisplayName: t.DisplayName,
			Subscribed:  subscribedIDs[t.ID],
		}
	}

	data := map[string]any{"topics": topicData, "saved": saved}
	rendered, err := h.renderer.RenderHTML(preferencesTemplate, data)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "preferences: render failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to render preferences")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, rendered)
}

func (h *PublicHandler) handleSavePreferences(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")

	user, err := h.users.GetUserByUnsubscribeToken(r.Context(), token)
	if err != nil {
		h.logger.ErrorContext(r.Context(), "save preferences: user lookup failed", slog.Any("error", err))
		writeError(w, http.StatusNotFound, "invalid token")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	topicIDStrs := r.Form["topic"]
	topicIDs := make([]uint, 0, len(topicIDStrs))
	for _, s := range topicIDStrs {
		id, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			continue
		}
		topicIDs = append(topicIDs, uint(id))
	}

	if err := h.preferences.SetUserTopics(r.Context(), user.MailingListName, user.ID, topicIDs); err != nil {
		h.logger.ErrorContext(r.Context(), "save preferences failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "failed to save preferences")
		return
	}

	listName := r.PathValue("listName")

	http.Redirect(w, r, fmt.Sprintf("/preferences/%s/%s?saved=1", listName, token), http.StatusSeeOther)
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
