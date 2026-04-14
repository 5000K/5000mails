package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

// SubscriptionService manages user subscriptions to mailing lists.
type SubscriptionService struct {
	lists         domain.MailingListRepository
	users         domain.UserRepository
	confirmations domain.ConfirmationRepository
	renderer      domain.Renderer
	sender        domain.Sender
	confirmMail   string // raw markdown template for the confirmation mail
	baseURL       string
}

// NewSubscriptionService creates a new SubscriptionService.
func NewSubscriptionService(
	lists domain.MailingListRepository,
	users domain.UserRepository,
	confirmations domain.ConfirmationRepository,
	renderer domain.Renderer,
	sender domain.Sender,
	confirmMail string,
	baseURL string,
) *SubscriptionService {
	return &SubscriptionService{
		lists:         lists,
		users:         users,
		confirmations: confirmations,
		renderer:      renderer,
		sender:        sender,
		confirmMail:   confirmMail,
		baseURL:       baseURL,
	}
}

// Subscribe adds a user to the mailing list with the given name and sends a
// confirmation mail to the user's address.
// Returns an error if the mailing list does not exist.
func (s *SubscriptionService) Subscribe(ctx context.Context, listName, userName, email string) (*domain.User, error) {
	list, err := s.lists.GetListByName(ctx, listName)
	if err != nil {
		return nil, fmt.Errorf("mailing list %q not found: %w", listName, err)
	}

	unsubToken, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating unsubscribe token: %w", err)
	}

	user, err := s.users.AddUser(ctx, list.Name, userName, email, unsubToken)
	if err != nil {
		return nil, fmt.Errorf("adding user to list %q: %w", listName, err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating confirmation token: %w", err)
	}

	if _, err := s.confirmations.CreateConfirmation(ctx, user.ID, token); err != nil {
		return nil, fmt.Errorf("creating confirmation for user %d: %w", user.ID, err)
	}

	metadata, body, err := s.renderer.Render(&s.confirmMail, map[string]any{
		"token":      token,
		"confirmURL": s.baseURL + "/confirm/" + token,
		"Recipient":  *user,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering confirmation mail: %w", err)
	}

	if err := s.sender.SendMail(ctx, metadata, body, *user); err != nil {
		return nil, fmt.Errorf("sending confirmation mail to %q: %w", email, err)
	}

	return user, nil
}

// Confirm completes the double opt-in for the confirmation identified by token.
func (s *SubscriptionService) Confirm(ctx context.Context, token string) error {
	confirmation, err := s.confirmations.GetConfirmationByToken(ctx, token)
	if err != nil {
		return fmt.Errorf("looking up confirmation token: %w", err)
	}

	if err := s.users.ConfirmUser(ctx, confirmation.UserID); err != nil {
		return fmt.Errorf("confirming user %d: %w", confirmation.UserID, err)
	}

	if err := s.confirmations.DeleteConfirmation(ctx, confirmation.ID); err != nil {
		return fmt.Errorf("deleting used confirmation %d: %w", confirmation.ID, err)
	}

	return nil
}

// Unsubscribe removes a user identified by their unsubscribe token.
func (s *SubscriptionService) Unsubscribe(ctx context.Context, unsubscribeToken string) error {
	user, err := s.users.GetUserByUnsubscribeToken(ctx, unsubscribeToken)
	if err != nil {
		return fmt.Errorf("user with unsubscribe token not found: %w", err)
	}

	if err := s.users.RemoveUser(ctx, user.ID); err != nil {
		return fmt.Errorf("removing user %d: %w", user.ID, err)
	}

	return nil
}

// generateToken returns a cryptographically random 32-byte hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
