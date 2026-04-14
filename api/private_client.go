package api

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// PrivateClient is an HTTP client for the private admin API.
// Supply a non-nil privateKey to have every request signed automatically.
type PrivateClient struct {
	baseURL    string
	privateKey ed25519.PrivateKey
	httpClient *http.Client
}

// NewPrivateClient creates a new PrivateClient.
// Pass a nil privateKey when the server has authentication disabled.
func NewPrivateClient(baseURL string, privateKey ed25519.PrivateKey) *PrivateClient {
	return &PrivateClient{
		baseURL:    baseURL,
		privateKey: privateKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ListResponse is returned by list creation and rename endpoints.
type ListResponse struct {
	Name string `json:"name"`
}

// ListDetailResponse is returned by the get-list endpoint.
type ListDetailResponse struct {
	Name        string `json:"name"`
	Subscribers struct {
		Total     int `json:"total"`
		Confirmed int `json:"confirmed"`
	} `json:"subscribers"`
}

// UserItem describes a single subscriber as returned by the get-users endpoint.
type UserItem struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Confirmed bool   `json:"confirmed"`
}

// RecipientInput is the test-mail recipient payload.
type RecipientInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetAllLists returns all mailing lists.
func (c *PrivateClient) GetAllLists(ctx context.Context) ([]ListResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, "/lists", nil)
	if err != nil {
		return nil, fmt.Errorf("get all lists: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get all lists: %w", err)
	}

	var out []ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("get all lists: decode response: %w", err)
	}
	return out, nil
}

// CreateList creates a new mailing list.
func (c *PrivateClient) CreateList(ctx context.Context, name string) (*ListResponse, error) {
	resp, err := c.do(ctx, http.MethodPost, "/lists", map[string]string{"name": name})
	if err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}

	var out ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("create list: decode response: %w", err)
	}
	return &out, nil
}

// GetList returns the mailing list with the given name along with subscriber stats.
func (c *PrivateClient) GetList(ctx context.Context, name string) (*ListDetailResponse, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/lists/%s", name), nil)
	if err != nil {
		return nil, fmt.Errorf("get list: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get list: %w", err)
	}

	var out ListDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("get list: decode response: %w", err)
	}
	return &out, nil
}

// RenameList renames the mailing list identified by name.
func (c *PrivateClient) RenameList(ctx context.Context, name, newName string) (*ListResponse, error) {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/lists/%s", name), map[string]string{"name": newName})
	if err != nil {
		return nil, fmt.Errorf("rename list: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("rename list: %w", err)
	}

	var out ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("rename list: decode response: %w", err)
	}
	return &out, nil
}

// DeleteList deletes the mailing list with the given name.
func (c *PrivateClient) DeleteList(ctx context.Context, name string) error {
	resp, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/lists/%s", name), nil)
	if err != nil {
		return fmt.Errorf("delete list: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("delete list: %w", err)
	}
	return nil
}

// GetUsers returns all subscribers (confirmed or not) for the named list.
func (c *PrivateClient) GetUsers(ctx context.Context, listName string) ([]UserItem, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/lists/%s/users", listName), nil)
	if err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get users: %w", err)
	}

	var out []UserItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("get users: decode response: %w", err)
	}
	return out, nil
}

// SendToList dispatches a rendered markdown mail to all confirmed subscribers
// of the named list.
func (c *PrivateClient) SendToList(ctx context.Context, listName string, raw string, data map[string]any) error {
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/lists/%s/send", listName), map[string]any{"raw": raw, "data": data})
	if err != nil {
		return fmt.Errorf("send to list: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return fmt.Errorf("send to list: %w", err)
	}
	return nil
}

// SendTestMail dispatches a rendered markdown mail to a single arbitrary recipient.
func (c *PrivateClient) SendTestMail(ctx context.Context, recipient RecipientInput, raw string, data map[string]any) error {
	payload := map[string]any{
		"recipient": recipient,
		"raw":       raw,
		"data":      data,
	}
	resp, err := c.do(ctx, http.MethodPost, "/mail/test", payload)
	if err != nil {
		return fmt.Errorf("send test mail: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return fmt.Errorf("send test mail: %w", err)
	}
	return nil
}

// ScheduledMailItem describes a single scheduled mail as returned by the API.
type ScheduledMailItem struct {
	ID              uint   `json:"id"`
	MailingListName string `json:"mailingListName"`
	ScheduledAt     int64  `json:"scheduledAt"`
	SentAt          *int64 `json:"sentAt"`
}

// ScheduleMail creates a new scheduled mail for the given list.
// scheduledAt is a unix timestamp (UTC).
func (c *PrivateClient) ScheduleMail(ctx context.Context, listName string, raw string, scheduledAt int64) (*ScheduledMailItem, error) {
	payload := map[string]any{"raw": raw, "scheduledAt": scheduledAt}
	resp, err := c.do(ctx, http.MethodPost, fmt.Sprintf("/lists/%s/schedule", listName), payload)
	if err != nil {
		return nil, fmt.Errorf("schedule mail: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("schedule mail: %w", err)
	}
	var out ScheduledMailItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("schedule mail: decode response: %w", err)
	}
	return &out, nil
}

// GetAllScheduled returns all scheduled mails.
func (c *PrivateClient) GetAllScheduled(ctx context.Context) ([]ScheduledMailItem, error) {
	resp, err := c.do(ctx, http.MethodGet, "/scheduled", nil)
	if err != nil {
		return nil, fmt.Errorf("get all scheduled: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get all scheduled: %w", err)
	}
	var out []ScheduledMailItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("get all scheduled: decode response: %w", err)
	}
	return out, nil
}

// GetScheduled returns a single scheduled mail by ID.
func (c *PrivateClient) GetScheduled(ctx context.Context, id uint) (*ScheduledMailItem, error) {
	resp, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/scheduled/%d", id), nil)
	if err != nil {
		return nil, fmt.Errorf("get scheduled: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("get scheduled: %w", err)
	}
	var out ScheduledMailItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("get scheduled: decode response: %w", err)
	}
	return &out, nil
}

// DeleteScheduled deletes a scheduled mail by ID.
func (c *PrivateClient) DeleteScheduled(ctx context.Context, id uint) error {
	resp, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/scheduled/%d", id), nil)
	if err != nil {
		return fmt.Errorf("delete scheduled: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusNoContent); err != nil {
		return fmt.Errorf("delete scheduled: %w", err)
	}
	return nil
}

// RescheduleMail changes the delivery time of a scheduled mail.
// scheduledAt is a unix timestamp (UTC).
func (c *PrivateClient) RescheduleMail(ctx context.Context, id uint, scheduledAt int64) (*ScheduledMailItem, error) {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/scheduled/%d/schedule", id), map[string]any{"scheduledAt": scheduledAt})
	if err != nil {
		return nil, fmt.Errorf("reschedule mail: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("reschedule mail: %w", err)
	}
	var out ScheduledMailItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("reschedule mail: decode response: %w", err)
	}
	return &out, nil
}

// ReplaceScheduledContent replaces the markdown body of a scheduled mail.
func (c *PrivateClient) ReplaceScheduledContent(ctx context.Context, id uint, raw string) (*ScheduledMailItem, error) {
	resp, err := c.do(ctx, http.MethodPut, fmt.Sprintf("/scheduled/%d/content", id), map[string]any{"raw": raw})
	if err != nil {
		return nil, fmt.Errorf("replace scheduled content: %w", err)
	}
	defer resp.Body.Close()

	if err := expectStatus(resp, http.StatusOK); err != nil {
		return nil, fmt.Errorf("replace scheduled content: %w", err)
	}
	var out ScheduledMailItem
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("replace scheduled content: decode response: %w", err)
	}
	return &out, nil
}

// do builds and executes a signed HTTP request.
func (c *PrivateClient) do(ctx context.Context, method, path string, payload any) (*http.Response, error) {
	var bodyBytes []byte
	if payload != nil {
		var err error
		bodyBytes, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if len(c.privateKey) > 0 {
		ts := strconv.FormatInt(time.Now().Unix(), 10)
		msg := buildSignedMessage(ts, method, path, bodyBytes)
		sig := ed25519.Sign(c.privateKey, msg)
		req.Header.Set("X-Timestamp", ts)
		req.Header.Set("X-Signature", hex.EncodeToString(sig))
	}

	return c.httpClient.Do(req)
}

func expectStatus(resp *http.Response, expected int) error {
	if resp.StatusCode == expected {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
