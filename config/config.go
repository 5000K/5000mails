package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

type TLSPolicy string

const (
	TLSMandatory     TLSPolicy = "TLSMandatory"
	TLSOpportunistic TLSPolicy = "TLSOpportunistic"
	NoTLS            TLSPolicy = "NoTLS"
)

type SmtpConfig struct {
	Host        string    `env:"SMTP_HOST" yaml:"host"`
	Port        int       `env:"SMTP_PORT" env-default:"587" yaml:"port"`
	Username    string    `env:"SMTP_USERNAME" yaml:"username"`
	Password    string    `env:"SMTP_PASSWORD" yaml:"password"`
	SenderEmail string    `env:"SMTP_SENDER_EMAIL" yaml:"sender-email"`
	TLSPolicy   TLSPolicy `env:"SMTP_TLS_POLICY" env-default:"TLSOpportunistic" yaml:"tls-policy"`
}

type MessageStrings struct {
	SubscribeSuccess                string `yaml:"subscribe-success"`
	SubscribeErrorInvalidInput      string `yaml:"subscribe-error-invalid-input"`
	SubscribeErrorAlreadySubscribed string `yaml:"subscribe-error-already-subscribed"`
	SubscribeError                  string `yaml:"subscribe-error"`
	ConfirmSuccess                  string `yaml:"confirm-success"`
	ConfirmErrorInvalidToken        string `yaml:"confirm-error-invalid-token"`
	UnsubscribeSuccess              string `yaml:"unsubscribe-success"`
	UnsubscribeErrorInvalidToken    string `yaml:"unsubscribe-error-invalid-token"`
	NewsletterNotFound              string `yaml:"newsletter-not-found"`
	PreferencesErrorInvalidToken    string `yaml:"preferences-error-invalid-token"`
	PreferencesError                string `yaml:"preferences-error"`
}

type Config struct {
	PublicAddr  string `env:"PUBLIC_ADDR" env-default:":8080" yaml:"public-addr"`
	PrivateAddr string `env:"PRIVATE_ADDR" env-default:":9000" yaml:"private-addr"`
	BaseURL     string `env:"BASE_URL" env-default:"http://localhost:8080" yaml:"base-url"`

	Smtp SmtpConfig `yaml:"smtp"`

	DB struct {
		Type string `env:"DB_TYPE" env-default:"sqlite" yaml:"type"`
		DSN  string `env:"DB_DSN" env-default:"5000mails.db" yaml:"dsn"`
	} `yaml:"db"`

	Auth struct {
		PublicKeyPath string `env:"AUTH_PUBLIC_KEY_PATH" yaml:"public-key-path"`
	} `yaml:"auth"`

	Strings MessageStrings `yaml:"strings"`

	Paths struct {
		Config      string `env:"CONFIG_PATH" env-default:"config.yml"`
		Template    string `env:"TEMPLATE_PATH" env-default:"https://github.com/5000K/5000mails/releases/latest/download/template.html" yaml:"template"`
		ConfirmMail string `env:"CONFIRM_MAIL_PATH" env-default:"https://github.com/5000K/5000mails/releases/latest/download/confirm.md" yaml:"confirm-mail"`
	} `yaml:"paths"`
}

// FetchResource reads a file from disk or downloads it over HTTP/HTTPS.
func FetchResource(urlOrPath string) ([]byte, error) {
	if strings.HasPrefix(urlOrPath, "http://") || strings.HasPrefix(urlOrPath, "https://") {
		resp, err := http.Get(urlOrPath) //nolint:noctx
		if err != nil {
			return nil, fmt.Errorf("fetch %q: %w", urlOrPath, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("fetch %q: HTTP %d", urlOrPath, resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch %q: read body: %w", urlOrPath, err)
		}
		return data, nil
	}
	data, err := os.ReadFile(urlOrPath)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", urlOrPath, err)
	}
	return data, nil
}

func Get() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	data, err := FetchResource(cfg.Paths.Config)
	if err != nil {
		return &cfg, nil
	}

	if err := cleanenv.ParseYAML(bytes.NewReader(data), &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
