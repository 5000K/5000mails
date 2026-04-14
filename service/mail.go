package service

import (
	"context"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

// MailService renders markdown content and dispatches it to mailing list
// recipients or arbitrary test addresses.
type MailService struct {
	lists       domain.MailingListRepository
	users       domain.UserRepository
	newsletters domain.SentNewsletterRepository
	renderer    domain.Renderer
	sender      domain.Sender
	baseURL     string
}

// NewMailService creates a new MailService.
func NewMailService(lists domain.MailingListRepository, users domain.UserRepository, newsletters domain.SentNewsletterRepository, renderer domain.Renderer, sender domain.Sender, baseURL string) *MailService {
	return &MailService{
		lists:       lists,
		users:       users,
		newsletters: newsletters,
		renderer:    renderer,
		sender:      sender,
		baseURL:     baseURL,
	}
}

// SendToList renders raw and sends the resulting mail to every confirmed
// subscriber of the mailing list identified by listName.
// data is passed through to the renderer as template variables.
func (s *MailService) SendToList(ctx context.Context, listName string, raw string, data map[string]any) error {
	list, err := s.lists.GetListByName(ctx, listName)
	if err != nil {
		return fmt.Errorf("looking up list %q: %w", listName, err)
	}

	recipients, err := s.users.GetConfirmedUsers(ctx, list.Name)
	if err != nil {
		return fmt.Errorf("getting confirmed users for list %q: %w", listName, err)
	}

	if len(recipients) == 0 {
		return nil
	}

	var firstMetadata domain.MailMetadata
	recipientIDs := make([]uint, 0, len(recipients))

	for i, recipient := range recipients {
		recipientData := make(map[string]any, len(data)+2)
		for k, v := range data {
			recipientData[k] = v
		}
		recipientData["Recipient"] = recipient
		recipientData["unsubscribeURL"] = s.baseURL + "/unsubscribe/" + recipient.UnsubscribeToken

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

	if _, err := s.newsletters.CreateSentNewsletter(ctx, firstMetadata.Subject, firstMetadata.SenderName, raw, recipientIDs, []string{listName}); err != nil {
		return fmt.Errorf("archiving sent newsletter: %w", err)
	}

	return nil
}

// SendTestMail renders raw and sends the resulting mail to the given user.
// The user is passed in directly and is not looked up from the database,
// making this suitable for previewing a newsletter before a real dispatch.
// data is passed through to the renderer as template variables.
func (s *MailService) SendTestMail(ctx context.Context, recipient domain.User, raw string, data map[string]any) error {
	recipientData := make(map[string]any, len(data)+2)
	for k, v := range data {
		recipientData[k] = v
	}
	recipientData["Recipient"] = recipient
	recipientData["unsubscribeURL"] = s.baseURL + "/unsubscribe/" + recipient.UnsubscribeToken

	metadata, body, err := s.renderer.Render(&raw, recipientData)
	if err != nil {
		return fmt.Errorf("rendering test mail for %q: %w", recipient.Email, err)
	}

	if err := s.sender.SendMail(ctx, metadata, body, recipient); err != nil {
		return fmt.Errorf("sending test mail to %q: %w", recipient.Email, err)
	}

	return nil
}

// AllNewsletters returns all archived sent newsletters.
func (s *MailService) AllNewsletters(ctx context.Context) ([]domain.SentNewsletter, error) {
	newsletters, err := s.newsletters.GetAllSentNewsletters(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sent newsletters: %w", err)
	}
	return newsletters, nil
}

// GetNewsletter returns a single archived newsletter by ID including recipients and mailing lists.
func (s *MailService) GetNewsletter(ctx context.Context, id uint) (*domain.SentNewsletter, error) {
	newsletter, err := s.newsletters.GetSentNewsletterByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting sent newsletter %d: %w", id, err)
	}
	return newsletter, nil
}

// DeleteNewsletter removes an archived newsletter by ID.
func (s *MailService) DeleteNewsletter(ctx context.Context, id uint) error {
	if err := s.newsletters.DeleteSentNewsletter(ctx, id); err != nil {
		return fmt.Errorf("deleting sent newsletter %d: %w", id, err)
	}
	return nil
}
