package service

import (
	"context"
	"fmt"

	"github.com/5000K/5000mails/domain"
)

// MailService renders markdown content and dispatches it to mailing list
// recipients or arbitrary test addresses.
type MailService struct {
	lists    domain.MailingListRepository
	users    domain.UserRepository
	renderer domain.Renderer
	sender   domain.Sender
	baseURL  string
}

// NewMailService creates a new MailService.
func NewMailService(lists domain.MailingListRepository, users domain.UserRepository, renderer domain.Renderer, sender domain.Sender, baseURL string) *MailService {
	return &MailService{
		lists:    lists,
		users:    users,
		renderer: renderer,
		sender:   sender,
		baseURL:  baseURL,
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

	for _, recipient := range recipients {
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

		if err := s.sender.SendMail(ctx, metadata, body, recipient); err != nil {
			return fmt.Errorf("sending mail to %q: %w", recipient.Email, err)
		}
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
