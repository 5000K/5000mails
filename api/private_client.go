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
