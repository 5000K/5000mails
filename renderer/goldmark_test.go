package renderer

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/5000K/5000mails/domain"
)

// layout that exposes both the html body and both metadata fields
const testLayout = `Subject:{{.metadata.Subject}} Sender:{{.metadata.SenderName}}
{{.html}}`

func newRenderer(t *testing.T) *GoldmarkRenderer {
	t.Helper()
	r, err := NewGoldmarkRenderer([]byte(testLayout), nil, slog.Default())
	if err != nil {
		t.Fatalf("NewGoldmarkRenderer: %v", err)
	}
	return r
}

// ---------- parseFrontmatter unit tests ----------

func TestParseFrontmatter_ValidBlock(t *testing.T) {
	input := "---\nsubject: \"Hello\"\nsender: \"Bot\"\n---\n# Body"
	meta, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Subject != "Hello" {
		t.Errorf("expected Subject %q, got %q", "Hello", meta.Subject)
	}
	if meta.SenderName != "Bot" {
		t.Errorf("expected SenderName %q, got %q", "Bot", meta.SenderName)
	}
	if !strings.HasPrefix(body, "# Body") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	input := "# Just markdown"
	meta, body, err := parseFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != (domain.MailMetadata{}) {
		t.Errorf("expected empty metadata, got %+v", meta)
	}
	if body != input {
		t.Errorf("expected body to equal input, got %q", body)
	}
}

func TestParseFrontmatter_UnclosedMarkerErrors(t *testing.T) {
	input := "---\nsubject: oops\n"
	_, _, err := parseFrontmatter(input)
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter, got nil")
	}
}

func TestParseFrontmatter_InvalidYAMLErrors(t *testing.T) {
	input := "---\n: bad: yaml: [\n---\n# body"
	_, _, err := parseFrontmatter(input)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

// ---------- GoldmarkRenderer.Render ----------

func TestRender_MetadataExtracted(t *testing.T) {
	r := newRenderer(t)
	raw := "---\nsubject: \"Newsletter\"\nsender: \"Alice\"\n---\nHello."
	meta, _, err := r.Render(&raw, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Subject != "Newsletter" {
		t.Errorf("expected Subject %q, got %q", "Newsletter", meta.Subject)
	}
	if meta.SenderName != "Alice" {
		t.Errorf("expected SenderName %q, got %q", "Alice", meta.SenderName)
	}
}

func TestRender_MarkdownConvertedToHTML(t *testing.T) {
	r := newRenderer(t)
	raw := "---\nsubject: \"S\"\nsender: \"B\"\n---\n**bold**"
	_, body, err := r.Render(&raw, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "<strong>bold</strong>") {
		t.Errorf("expected <strong>bold</strong> in body, got:\n%s", body)
	}
}

func TestRender_ContentTemplatingApplied(t *testing.T) {
	r := newRenderer(t)
	raw := "---\nsubject: \"S\"\nsender: \"B\"\n---\nHello, {{.name}}!"
	data := map[string]any{"name": "World"}
	_, body, err := r.Render(&raw, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Hello, World!") {
		t.Errorf("expected templated name in output, got:\n%s", body)
	}
}

func TestRender_LayoutTemplateReceivesHTMLAndMetadata(t *testing.T) {
	r := newRenderer(t)
	raw := "---\nsubject: \"Weekly\"\nsender: \"Bot\"\n---\n# Hi"
	meta, body, err := r.Render(&raw, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "Subject:"+meta.Subject) {
		t.Errorf("expected Subject in body, got:\n%s", body)
	}
	if !strings.Contains(body, "Sender:"+meta.SenderName) {
		t.Errorf("expected SenderName in body, got:\n%s", body)
	}
}

func TestRender_InvalidContentTemplateErrors(t *testing.T) {
	r := newRenderer(t)
	raw := "---\nsubject: S\nsender: B\n---\n{{.unclosed"
	_, _, err := r.Render(&raw, nil)
	if err == nil {
		t.Fatal("expected error for invalid content template, got nil")
	}
}

func TestRender_InvalidLayoutTemplateErrors(t *testing.T) {
	_, err := NewGoldmarkRenderer([]byte("{{.unclosed"), nil, slog.Default())
	if err == nil {
		t.Fatal("expected error for invalid layout template, got nil")
	}
}

func TestRender_ThemeInjectedIntoLayout(t *testing.T) {
	layout := `<style>{{.theme}}</style>{{.html}}`
	r, err := NewGoldmarkRenderer([]byte(layout), []byte("body{color:red}"), slog.Default())
	if err != nil {
		t.Fatalf("NewGoldmarkRenderer: %v", err)
	}
	raw := "---\nsubject: S\nsender: B\n---\nhi"
	_, body, err := r.Render(&raw, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(body, "<style>body{color:red}</style>") {
		t.Errorf("expected theme in layout output, got:\n%s", body)
	}
}

func TestRender_ExtraDataPassedToLayout(t *testing.T) {
	layout := `{{.customKey}}: {{.html}}`
	r, err := NewGoldmarkRenderer([]byte(layout), nil, slog.Default())
	if err != nil {
		t.Fatalf("NewGoldmarkRenderer: %v", err)
	}
	raw := "---\nsubject: S\nsender: B\n---\nhi"
	data := map[string]any{"customKey": "injected"}
	_, body, err := r.Render(&raw, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(body, "injected:") {
		t.Errorf("expected customKey in layout output, got:\n%s", body)
	}
}
