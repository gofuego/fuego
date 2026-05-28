package parse

import (
	"context"
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/discover"
	"os"
	"path/filepath"
)

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEnv     bool
		wantPayload string
		wantErr     bool
	}{
		{
			name:        "standard frontmatter",
			input:       "---\ntitle: Hello\n---\nBody content",
			wantEnv:     true,
			wantPayload: "Body content",
		},
		{
			name:        "no frontmatter",
			input:       "Just plain text",
			wantEnv:     false,
			wantPayload: "Just plain text",
		},
		{
			name:    "unclosed frontmatter",
			input:   "---\ntitle: Hello\nBody content",
			wantErr: true,
		},
		{
			name:        "empty payload",
			input:       "---\ntitle: Hello\n---\n",
			wantEnv:     true,
			wantPayload: "",
		},
		{
			name:        "frontmatter with multiple fields",
			input:       "---\ntitle: Test\nlayout: quiz\ntags:\n  - go\n  - web\n---\nPayload here",
			wantEnv:     true,
			wantPayload: "Payload here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, payload, err := SplitFrontmatter([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantEnv && len(env) == 0 {
				t.Error("expected non-empty envelope")
			}
			if !tt.wantEnv && len(env) != 0 {
				t.Errorf("expected empty envelope, got %v", env)
			}

			if string(payload) != tt.wantPayload {
				t.Errorf("payload: got %q, want %q", string(payload), tt.wantPayload)
			}
		})
	}
}

func TestSplitFrontmatterFields(t *testing.T) {
	input := "---\ntitle: Hello World\nlayout: quiz\npoints: 10\n---\nbody"
	env, _, err := SplitFrontmatter([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["title"] != "Hello World" {
		t.Errorf("expected title 'Hello World', got %v", env["title"])
	}
	if env["layout"] != "quiz" {
		t.Errorf("expected layout 'quiz', got %v", env["layout"])
	}
	if env["points"] != 10 {
		t.Errorf("expected points 10, got %v", env["points"])
	}
}

// mockParser for testing
type testParser struct {
	typ string
}

func (p *testParser) Type() string { return p.typ }
func (p *testParser) Parse(raw []byte, meta core.Envelope) ([]core.Node, error) {
	return []core.Node{{Type: p.typ, Content: string(raw)}}, nil
}

func writeTestFile(t *testing.T, dir, relPath, content string) string {
	t.Helper()
	full := filepath.Join(dir, relPath)
	os.MkdirAll(filepath.Dir(full), 0755)
	os.WriteFile(full, []byte(content), 0644)
	return full
}

func TestParseAllRawPassthrough(t *testing.T) {
	dir := t.TempDir()
	absPath := writeTestFile(t, dir, "hello.md", "---\ntitle: Hello\n---\nSome content")

	files := []discover.FileEntry{
		{Path: absPath, RelPath: "hello.md", Ext: "md"},
	}

	pages, errs := ParseAll(context.Background(), files, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	p := pages[0]
	if p.Envelope["title"] != "Hello" {
		t.Errorf("expected title 'Hello', got %v", p.Envelope["title"])
	}
	if !p.IsRaw {
		t.Error("expected raw passthrough")
	}
	if len(p.Nodes) != 1 || p.Nodes[0].Type != "raw" {
		t.Errorf("expected single 'raw' node, got %v", p.Nodes)
	}
	if p.Nodes[0].Content != "Some content" {
		t.Errorf("expected content 'Some content', got %q", p.Nodes[0].Content)
	}
}

func TestParseAllWithParser(t *testing.T) {
	dir := t.TempDir()
	absPath := writeTestFile(t, dir, "q1.trivia", "---\ntitle: Nash\ntype: trivia\n---\nQuestion text")

	files := []discover.FileEntry{
		{Path: absPath, RelPath: "q1.trivia", Ext: "trivia"},
	}

	parsers := map[string]core.Parser{
		"trivia": &testParser{typ: "trivia"},
	}

	pages, errs := ParseAll(context.Background(), files, parsers)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}

	p := pages[0]
	if p.IsRaw {
		t.Error("should not be raw when parser exists")
	}
	if p.Type != "trivia" {
		t.Errorf("expected type 'trivia', got %q", p.Type)
	}
	if len(p.Nodes) != 1 || p.Nodes[0].Type != "trivia" {
		t.Errorf("expected trivia node, got %v", p.Nodes)
	}
}

func TestParseAllTypeOverride(t *testing.T) {
	dir := t.TempDir()
	absPath := writeTestFile(t, dir, "q1.card", "---\ntitle: Test\ntype: special\n---\nPayload")

	files := []discover.FileEntry{
		{Path: absPath, RelPath: "q1.card", Ext: "card"},
	}

	parsers := map[string]core.Parser{
		"special": &testParser{typ: "special"},
	}

	pages, errs := ParseAll(context.Background(), files, parsers)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if pages[0].Type != "special" {
		t.Errorf("expected type override to 'special', got %q", pages[0].Type)
	}
	if pages[0].IsRaw {
		t.Error("should use 'special' parser, not raw passthrough")
	}
}

func TestParseAllLayoutFromEnvelope(t *testing.T) {
	dir := t.TempDir()
	absPath := writeTestFile(t, dir, "q1.md", "---\ntitle: Test\nlayout: quiz\n---\nBody")

	files := []discover.FileEntry{
		{Path: absPath, RelPath: "q1.md", Ext: "md"},
	}

	pages, _ := ParseAll(context.Background(), files, nil)
	if pages[0].Layout != "quiz" {
		t.Errorf("expected layout 'quiz', got %q", pages[0].Layout)
	}
}

func TestParseAllBadFrontmatter(t *testing.T) {
	dir := t.TempDir()
	absPath := writeTestFile(t, dir, "bad.md", "---\ntitle: Hello\nBody without closing")

	files := []discover.FileEntry{
		{Path: absPath, RelPath: "bad.md", Ext: "md"},
	}

	pages, errs := ParseAll(context.Background(), files, nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Severity != core.LocalFatal {
		t.Errorf("expected LocalFatal, got %v", errs[0].Severity)
	}
	if len(pages) != 0 {
		t.Errorf("expected 0 pages, got %d", len(pages))
	}
}

func TestParseAllMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	path1 := writeTestFile(t, dir, "a.md", "---\ntitle: A\n---\nContent A")
	path2 := writeTestFile(t, dir, "b.md", "---\ntitle: B\n---\nContent B")
	path3 := writeTestFile(t, dir, "c.md", "---\ntitle: C\n---\nContent C")

	files := []discover.FileEntry{
		{Path: path1, RelPath: "a.md", Ext: "md"},
		{Path: path2, RelPath: "b.md", Ext: "md"},
		{Path: path3, RelPath: "c.md", Ext: "md"},
	}

	pages, errs := ParseAll(context.Background(), files, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
}
