package service

import (
	"context"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

type MailService struct {
	lists       domain.MailingListRepository
	users       domain.UserRepository
	topics      domain.TopicRepository
	newsletters domain.SentNewsletterRepository
	renderer    domain.Renderer
	sender      domain.Sender
	baseURL     string
}

func NewMailService(lists domain.MailingListRepository, users domain.UserRepository, topics domain.TopicRepository, newsletters domain.SentNewsletterRepository, renderer domain.Renderer, sender domain.Sender, baseURL string) *MailService {
	return &MailService{
		lists:       lists,
		users:       users,
		topics:      topics,
		newsletters: newsletters,
		renderer:    renderer,
		sender:      sender,
		baseURL:     baseURL,
	}
}

func (s *MailService) SendToList(ctx context.Context, listName string, raw string, topicNames []string, data map[string]any) error {
	list, err := s.lists.GetListByName(ctx, listName)
	if err != nil {
		return fmt.Errorf("looking up list %q: %w", listName, err)
	}

	var recipients []domain.User
	if len(topicNames) > 0 {
		recipients, err = s.topics.GetConfirmedUsersSubscribedToTopics(ctx, list.Name, topicNames)
	} else {
		recipients, err = s.users.GetConfirmedUsers(ctx, list.Name)
	}
	if err != nil {
		return fmt.Errorf("getting recipients for list %q: %w", listName, err)
	}

	if len(recipients) == 0 {
		return nil
	}

	var firstMetadata domain.MailMetadata
	recipientIDs := make([]uint, 0, len(recipients))

	for i, recipient := range recipients {
		recipientData := make(map[string]any, len(data)+3)
		for k, v := range data {
			recipientData[k] = v
		}
		recipientData["Recipient"] = recipient
		recipientData["unsubscribeURL"] = s.baseURL + "/unsubscribe/" + recipient.UnsubscribeToken
		recipientData["preferencesURL"] = s.baseURL + "/preferences/" + listName + "/" + recipient.UnsubscribeToken

		metadata, body, err := s.renderer.Render(&raw, recipientData)
		if err != nil {
			return fmt.Errorf("rendering mail for %q: %w", recipient.Email, err)
		}
		if i == 0 {
			firstMetadata = metadata
		}

		if err := s.sender.SendMail(ctx, metadata, body, recipient); err != nil {
			return fmt.Errorf("sending mail to %q: %w", recipient.Email, err)
		}
		recipientIDs = append(recipientIDs, recipient.ID)
	}

	if _, err := s.newsletters.CreateSentNewsletter(ctx, firstMetadata.Subject, firstMetadata.SenderName, raw, recipientIDs, []string{listName}, topicNames); err != nil {
		return fmt.Errorf("archiving sent newsletter: %w", err)
	}

	return nil
}

func (s *MailService) SendTestMail(ctx context.Context, recipient domain.User, raw string, data map[string]any) error {
	recipientData := make(map[string]any, len(data)+3)
	for k, v := range data {
		recipientData[k] = v
	}
	recipientData["Recipient"] = recipient
	recipientData["unsubscribeURL"] = s.baseURL + "/unsubscribe/" + recipient.UnsubscribeToken
	recipientData["preferencesURL"] = s.baseURL + "/preferences/" + recipient.MailingListName + "/" + recipient.UnsubscribeToken

	metadata, body, err := s.renderer.Render(&raw, recipientData)
	if err != nil {
		return fmt.Errorf("rendering test mail for %q: %w", recipient.Email, err)
	}

	if err := s.sender.SendMail(ctx, metadata, body, recipient); err != nil {
		return fmt.Errorf("sending test mail to %q: %w", recipient.Email, err)
	}

	return nil
}

func (s *MailService) AllNewsletters(ctx context.Context) ([]domain.SentNewsletter, error) {
	newsletters, err := s.newsletters.GetAllSentNewsletters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sent newsletters: %w", err)
	}
	return newsletters, nil
}

func (s *MailService) GetNewsletter(ctx context.Context, id uint) (*domain.SentNewsletter, error) {
	newsletter, err := s.newsletters.GetSentNewsletterByID(ctx, id, true)
	if err != nil {
		return nil, fmt.Errorf("getting sent newsletter %d: %w", id, err)
	}
	return newsletter, nil
}

func (s *MailService) DeleteNewsletter(ctx context.Context, id uint) error {
	if err := s.newsletters.DeleteSentNewsletter(ctx, id); err != nil {
		return fmt.Errorf("deleting sent newsletter %d: %w", id, err)
	}
	return nil
}

var placeholderUser = domain.User{Name: "Subscriber", Email: "you@example.com"}

func (s *MailService) RenderNewsletter(ctx context.Context, id uint, unsubscribeToken string) (string, error) {
	newsletter, err := s.newsletters.GetSentNewsletterByID(ctx, id, false)
	if err != nil {
		return "", fmt.Errorf("loading newsletter %d: %w", id, err)
	}

	recipient := placeholderUser
	if unsubscribeToken != "" {
		if u, err := s.users.GetUserByUnsubscribeToken(ctx, unsubscribeToken); err == nil {
			recipient = *u
		}
	}

	listName := recipient.MailingListName
	if listName == "" && len(newsletter.MailingLists) > 0 {
		listName = newsletter.MailingLists[0].Name
	}

	data := map[string]any{
		"Recipient":      recipient,
		"unsubscribeURL": s.baseURL + "/unsubscribe/" + recipient.UnsubscribeToken,
		"preferencesURL": s.baseURL + "/preferences/" + listName + "/" + recipient.UnsubscribeToken,
	}

	_, body, err := s.renderer.Render(&newsletter.RawMarkdown, data)
	if err != nil {
		return "", fmt.Errorf("rendering newsletter %d: %w", id, err)
	}
	return body, nil
}
