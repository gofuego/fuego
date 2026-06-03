package markdown

import (
	"strings"
	"testing"

	"github.com/FabioSol/fuego/core"
)

func TestParser_Type(t *testing.T) {
	p := Parser()
	if p.Type() != "md" {
		t.Errorf("expected type 'md', got %q", p.Type())
	}
}

func TestParser_BasicParagraph(t *testing.T) {
	p := Parser()
	raw := []byte("---\ntitle: Test\n---\nHello world.")
	env, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["title"] != "Test" {
		t.Errorf("expected title 'Test', got %v", env["title"])
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if !nodes[0].Raw {
		t.Error("expected Raw to be true")
	}
	if !strings.Contains(nodes[0].Content, "<p>Hello world.</p>") {
		t.Errorf("expected paragraph HTML, got %q", nodes[0].Content)
	}
}

func TestParser_GFMTable(t *testing.T) {
	p := Parser()
	raw := []byte("| A | B |\n|---|---|\n| 1 | 2 |\n")
	_, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(nodes[0].Content, "<table>") {
		t.Errorf("expected table, got %q", nodes[0].Content)
	}
}

func TestParser_EmptyPayload(t *testing.T) {
	p := Parser()
	raw := []byte("---\ntitle: Test\n---\n")
	_, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestParser_NoFrontmatter(t *testing.T) {
	p := Parser()
	raw := []byte("Hello world.")
	env, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty envelope, got %v", env)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestParser_Interface(t *testing.T) {
	var _ core.Parser = Parser()
}
