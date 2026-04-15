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
	topics        domain.TopicRepository
	renderer      domain.Renderer
	sender        domain.Sender
	confirmMail   string
	baseURL       string
}

func NewSubscriptionService(
	lists domain.MailingListRepository,
	users domain.UserRepository,
	confirmations domain.ConfirmationRepository,
	topics domain.TopicRepository,
	renderer domain.Renderer,
	sender domain.Sender,
	confirmMail string,
	baseURL string,
) *SubscriptionService {
	return &SubscriptionService{
		lists:         lists,
		users:         users,
		confirmations: confirmations,
		topics:        topics,
		renderer:      renderer,
		sender:        sender,
		confirmMail:   confirmMail,
		baseURL:       baseURL,
	}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, listName, userName, email string, topicNames []string) (*domain.User, error) {
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

	if err := s.subscribeToTopics(ctx, user.ID, list.Name, topicNames); err != nil {
		return nil, fmt.Errorf("subscribing user to topics: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generating confirmation token: %w", err)
	}

	if _, err := s.confirmations.CreateConfirmation(ctx, user.ID, token); err != nil {
		return nil, fmt.Errorf("creating confirmation for user %d: %w", user.ID, err)
	}

	metadata, body, err := s.renderer.Render(&s.confirmMail, map[string]any{
		"token":          token,
		"confirmURL":     s.baseURL + "/confirm/" + token,
		"preferencesURL": s.baseURL + "/preferences/" + listName + "/" + user.UnsubscribeToken,
		"Recipient":      *user,
	})
	if err != nil {
		return nil, fmt.Errorf("rendering confirmation mail: %w", err)
	}

	if err := s.sender.SendMail(ctx, metadata, body, *user); err != nil {
		return nil, fmt.Errorf("sending confirmation mail to %q: %w", email, err)
	}

	return user, nil
}

func (s *SubscriptionService) subscribeToTopics(ctx context.Context, userID uint, listName string, topicNames []string) error {
	var topics []domain.Topic
	var err error

	if len(topicNames) > 0 {
		allTopics, err := s.topics.GetTopicsByList(ctx, listName)
		if err != nil {
			return fmt.Errorf("getting topics for list %q: %w", listName, err)
		}
		nameSet := make(map[string]bool, len(topicNames))
		for _, n := range topicNames {
			nameSet[n] = true
		}
		for _, t := range allTopics {
			if nameSet[t.Name] {
				topics = append(topics, t)
			}
		}
	} else {
		topics, err = s.topics.GetDefaultEnabledTopics(ctx, listName)
		if err != nil {
			return fmt.Errorf("getting default topics for list %q: %w", listName, err)
		}
	}

	if len(topics) == 0 {
		return nil
	}

	topicIDs := make([]uint, len(topics))
	for i, t := range topics {
		topicIDs[i] = t.ID
	}
	return s.topics.SubscribeUserToTopics(ctx, userID, topicIDs)
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
