package smtp

import (
	"bytes"
	"context"
	"fmt"
	stdhtml "html"
	"log/slog"
	"strings"

	gomail "github.com/wneessen/go-mail"

	"github.com/5000K/5000mails/config"
	"github.com/5000K/5000mails/domain"
)

var blockElements = []string{
	"address", "article", "aside", "blockquote", "br", "dd", "details",
	"dialog", "div", "dl", "dt", "fieldset", "figcaption", "figure",
	"footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header",
	"hgroup", "hr", "li", "main", "nav", "ol", "p", "pre", "section",
	"summary", "table", "td", "th", "tr", "ul",
}

func htmlToPlainText(src []byte) []byte {
	var buf bytes.Buffer
	i := 0
	for i < len(src) {
		if src[i] != '<' {
			buf.WriteByte(src[i])
			i++
			continue
		}
		end := bytes.IndexByte(src[i:], '>')
		if end == -1 {
			buf.Write(src[i:])
			break
		}
		inner := src[i+1 : i+end]
		if len(inner) > 0 && inner[0] == '/' {
			inner = inner[1:]
		}
		tagName := strings.ToLower(string(inner))
		if sp := strings.IndexByte(tagName, ' '); sp != -1 {
			tagName = tagName[:sp]
		}
		for _, bt := range blockElements {
			if tagName == bt {
				buf.WriteByte('\n')
				break
			}
		}
		i += end + 1
	}
	text := stdhtml.UnescapeString(buf.String())
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return []byte(strings.TrimSpace(text))
}

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
	msg.AddAlternativeString(gomail.TypeTextPlain, string(htmlToPlainText([]byte(body))))

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
