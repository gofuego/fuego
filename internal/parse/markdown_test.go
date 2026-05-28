package parse

import (
	"strings"
	"testing"

	"github.com/FabioSol/fuego/core"
)

func TestMarkdownParser_Type(t *testing.T) {
	p := NewMarkdownParser()
	if p.Type() != "md" {
		t.Errorf("expected type 'md', got %q", p.Type())
	}
}

func TestMarkdownParser_BasicParagraph(t *testing.T) {
	p := NewMarkdownParser()
	nodes, err := p.Parse([]byte("Hello world."), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != "html" {
		t.Errorf("expected type 'html', got %q", nodes[0].Type)
	}
	if !strings.Contains(nodes[0].Content, "<p>Hello world.</p>") {
		t.Errorf("expected paragraph HTML, got %q", nodes[0].Content)
	}
}

func TestMarkdownParser_HeadingsAndFormatting(t *testing.T) {
	p := NewMarkdownParser()
	input := "# Title\n\nSome **bold** text.\n"
	nodes, err := p.Parse([]byte(input), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := nodes[0].Content
	if !strings.Contains(html, "<h1>Title</h1>") {
		t.Errorf("expected h1, got %q", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("expected strong, got %q", html)
	}
}

func TestMarkdownParser_GFMTable(t *testing.T) {
	p := NewMarkdownParser()
	input := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	nodes, err := p.Parse([]byte(input), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := nodes[0].Content
	if !strings.Contains(html, "<table>") {
		t.Errorf("expected table, got %q", html)
	}
}

func TestMarkdownParser_GFMStrikethrough(t *testing.T) {
	p := NewMarkdownParser()
	input := "~~deleted~~\n"
	nodes, err := p.Parse([]byte(input), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := nodes[0].Content
	if !strings.Contains(html, "<del>deleted</del>") {
		t.Errorf("expected strikethrough, got %q", html)
	}
}

func TestMarkdownParser_EmptyPayload(t *testing.T) {
	p := NewMarkdownParser()
	nodes, err := p.Parse([]byte(""), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for empty input, got %d", len(nodes))
	}
}

func TestMarkdownParser_CodeBlock(t *testing.T) {
	p := NewMarkdownParser()
	input := "```go\nfmt.Println(\"hello\")\n```\n"
	nodes, err := p.Parse([]byte(input), core.Envelope{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html := nodes[0].Content
	if !strings.Contains(html, `class="language-go"`) {
		t.Errorf("expected language class, got %q", html)
	}
}

func TestMarkdownParser_Interface(t *testing.T) {
	var _ core.Parser = NewMarkdownParser()
}

func TestBuiltinParsers_ContainsMd(t *testing.T) {
	builtins := BuiltinParsers()
	if _, ok := builtins["md"]; !ok {
		t.Error("expected 'md' in BuiltinParsers()")
	}
}
