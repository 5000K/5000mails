package renderer

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/5000K/5000mails/domain"
	"github.com/yuin/goldmark"
	"gopkg.in/yaml.v3"
)

// GoldmarkRenderer implements domain.Renderer using Go templates and Goldmark.
type GoldmarkRenderer struct {
	tmpl   *template.Template
	logger *slog.Logger
	md     goldmark.Markdown
}

// NewGoldmarkRenderer parses tmpl as a Go HTML template and returns a renderer.
func NewGoldmarkRenderer(tmpl []byte, logger *slog.Logger) (*GoldmarkRenderer, error) {
	t, err := template.New("layout").Parse(string(tmpl))
	if err != nil {
		return nil, fmt.Errorf("parsing renderer layout template: %w", err)
	}
	return &GoldmarkRenderer{
		tmpl:   t,
		logger: logger,
		md:     goldmark.New(),
	}, nil
}

// Render implements domain.Renderer.
//
// Pipeline:
//  1. Execute raw as a Go template with data.
//  2. Strip and parse the YAML frontmatter into MailMetadata.
//  3. Convert the remaining Markdown body to HTML via Goldmark.
//  4. Execute the layout template with data + "html" + "metadata" keys.
func (r *GoldmarkRenderer) Render(raw *string, data map[string]any) (domain.MailMetadata, string, error) {
	templated, err := applyTemplate("content", *raw, data)
	if err != nil {
		return domain.MailMetadata{}, "", fmt.Errorf("templating markdown content: %w", err)
	}

	metadata, rawFM, markdownBody, err := parseFrontmatter(templated)
	if err != nil {
		return domain.MailMetadata{}, "", fmt.Errorf("parsing frontmatter: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := r.md.Convert([]byte(markdownBody), &htmlBuf); err != nil {
		return domain.MailMetadata{}, "", fmt.Errorf("converting markdown to html: %w", err)
	}

	layoutData := mergeData(data, map[string]any{
		"html":        htmlBuf.String(),
		"metadata":    metadata,
		"frontmatter": rawFM,
	})

	var finalBuf bytes.Buffer
	if err := r.tmpl.Execute(&finalBuf, layoutData); err != nil {
		return domain.MailMetadata{}, "", fmt.Errorf("executing layout template: %w", err)
	}

	r.logger.Debug("rendered mail", slog.String("subject", metadata.Subject))
	return metadata, finalBuf.String(), nil
}

func (r *GoldmarkRenderer) RenderHTML(html string, data map[string]any) (string, error) {
	templated, err := applyTemplate("html-content", html, data)
	if err != nil {
		return "", fmt.Errorf("templating html content: %w", err)
	}

	layoutData := mergeData(data, map[string]any{
		"html":     templated,
		"metadata": domain.MailMetadata{},
	})

	var buf bytes.Buffer
	if err := r.tmpl.Execute(&buf, layoutData); err != nil {
		return "", fmt.Errorf("executing layout template for html: %w", err)
	}
	return buf.String(), nil
}

func applyTemplate(name, text string, data map[string]any) (string, error) {
	t, err := template.New(name).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", name, err)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %q: %w", name, err)
	}
	return buf.String(), nil
}

type frontmatterFields struct {
	Subject string `yaml:"subject"`
	Sender  string `yaml:"sender"`
}

func parseFrontmatter(s string) (domain.MailMetadata, map[string]any, string, error) {
	const marker = "---"
	if !strings.HasPrefix(s, marker) {
		return domain.MailMetadata{}, nil, s, nil
	}

	after := strings.TrimPrefix(s, marker)
	after = strings.TrimPrefix(after, "\r\n")
	after = strings.TrimPrefix(after, "\n")

	end := strings.Index(after, "\n---")
	if end == -1 {
		return domain.MailMetadata{}, nil, "", fmt.Errorf("frontmatter opening marker has no closing marker")
	}

	yamlSrc := after[:end]
	body := after[end+4:] // skip \n---
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")

	var fm frontmatterFields
	if err := yaml.Unmarshal([]byte(yamlSrc), &fm); err != nil {
		return domain.MailMetadata{}, nil, "", fmt.Errorf("parsing frontmatter yaml: %w", err)
	}

	var rawFM map[string]any
	if err := yaml.Unmarshal([]byte(yamlSrc), &rawFM); err != nil {
		return domain.MailMetadata{}, nil, "", fmt.Errorf("parsing frontmatter yaml: %w", err)
	}

	return domain.MailMetadata{Subject: fm.Subject, SenderName: fm.Sender}, rawFM, body, nil
}

func mergeData(base, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}
