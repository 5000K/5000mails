package smtp

import (
	"context"
	"fmt"
	"log/slog"

	gomail "github.com/wneessen/go-mail"

	"github.com/5000K/5000mails/config"
	"github.com/5000K/5000mails/domain"
)

type Sender struct {
	client      *gomail.Client
	senderEmail string
	logger      *slog.Logger
}

func tlsPolicy(p config.TLSPolicy) gomail.TLSPolicy {
	switch p {
	case config.TLSMandatory:
		return gomail.TLSMandatory
	case config.NoTLS:
		return gomail.NoTLS
	default:
		return gomail.TLSOpportunistic
	}
}

func NewSender(cfg config.SmtpConfig, logger *slog.Logger) (*Sender, error) {
	client, err := gomail.NewClient(
		cfg.Host,
		gomail.WithPort(cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(cfg.Username),
		gomail.WithPassword(cfg.Password),
		gomail.WithTLSPolicy(tlsPolicy(cfg.TLSPolicy)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating smtp client: %w", err)
	}

	return &Sender{
		client:      client,
		senderEmail: cfg.SenderEmail,
		logger:      logger,
	}, nil
}

func (s *Sender) SendMail(ctx context.Context, metadata domain.MailMetadata, body string, recipient domain.User) error {
	msg := gomail.NewMsg()

	if err := msg.FromFormat(metadata.SenderName, s.senderEmail); err != nil {
		return fmt.Errorf("setting from address: %w", err)
	}

	if err := msg.AddToFormat(recipient.Name, recipient.Email); err != nil {
		return fmt.Errorf("setting to address for %q: %w", recipient.Email, err)
	}

	msg.Subject(metadata.Subject)
	msg.SetBodyString(gomail.TypeTextHTML, body)

	if err := s.client.DialAndSendWithContext(ctx, msg); err != nil {
		s.logger.ErrorContext(ctx, "failed to send mail",
			slog.String("recipient", recipient.Email),
			slog.String("subject", metadata.Subject),
			slog.Any("error", err),
		)
		return fmt.Errorf("sending mail to %q: %w", recipient.Email, err)
	}

	s.logger.InfoContext(ctx, "mail sent",
		slog.String("recipient", recipient.Email),
		slog.String("subject", metadata.Subject),
	)

	return nil
}
