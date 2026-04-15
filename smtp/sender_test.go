package smtp

import (
	"testing"
)

func TestHtmlToPlainText(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "strips tags and keeps text",
			src:  "<p>Hello, <strong>world</strong>!</p>",
			want: "Hello, world!",
		},
		{
			name: "inserts newline at block boundaries",
			src:  "<p>First</p><p>Second</p>",
			want: "First\n\nSecond",
		},
		{
			name: "unescapes html entities",
			src:  "<p>a &amp; b &lt;3</p>",
			want: "a & b <3",
		},
		{
			name: "collapses excessive blank lines",
			src:  "<h1>Title</h1><br><br><p>Body</p>",
			want: "Title\n\nBody",
		},
		{
			name: "handles unclosed tag at end",
			src:  "<p>Truncated <b",
			want: "Truncated <b",
		},
		{
			name: "empty input",
			src:  "",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := string(htmlToPlainText([]byte(tc.src)))
			if got != tc.want {
				t.Errorf("htmlToPlainText(%q)\n got  %q\n want %q", tc.src, got, tc.want)
			}
		})
	}
}
