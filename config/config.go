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

type Config struct {
	// used as base url for the confirmation link in the confirmation mail.
	BaseURL string `env:"BASE_URL" env-default:":8080" yaml:"base-url"`

	Smtp SmtpConfig `yaml:"smtp"`

	Paths struct {
		Config string `env:"CONFIG_PATH" env-default:"config.yml"`

		Template string `env:"TEMPLATE_PATH" env-default:"./template.html" yaml:"template"`
		Theme    string `env:"THEME_PATH" env-default:"./theme.css" yaml:"theme"`
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
