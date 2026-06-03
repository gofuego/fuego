package core

import (
	"fmt"
	"strings"
	"testing"
)

func TestWithYAMLFrontmatter(t *testing.T) {
	p := WithYAMLFrontmatter("card", func(payload []byte, meta Envelope) ([]Node, error) {
		return []Node{{Type: "line", Content: strings.TrimSpace(string(payload))}}, nil
	})

	if p.Type() != "card" {
		t.Errorf("expected type 'card', got %q", p.Type())
	}

	raw := []byte("---\ntitle: Hello\nlayout: quiz\n---\nBody content")
	env, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env["title"] != "Hello" {
		t.Errorf("expected title 'Hello', got %v", env["title"])
	}
	if env["layout"] != "quiz" {
		t.Errorf("expected layout 'quiz', got %v", env["layout"])
	}
	if len(nodes) != 1 || nodes[0].Content != "Body content" {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

func TestWithYAMLFrontmatter_NoFrontmatter(t *testing.T) {
	p := WithYAMLFrontmatter("txt", func(payload []byte, meta Envelope) ([]Node, error) {
		return []Node{{Type: "raw", Content: string(payload)}}, nil
	})

	raw := []byte("Just plain text")
	env, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(env) != 0 {
		t.Errorf("expected empty envelope, got %v", env)
	}
	if len(nodes) != 1 || nodes[0].Content != "Just plain text" {
		t.Errorf("unexpected nodes: %v", nodes)
	}
}

func TestWithYAMLFrontmatter_MalformedYAML(t *testing.T) {
	p := WithYAMLFrontmatter("md", func(payload []byte, meta Envelope) ([]Node, error) {
		return nil, nil
	})

	raw := []byte("---\ntitle: Hello\nBody without closing")
	_, _, err := p.Parse(raw)
	if err == nil {
		t.Fatal("expected error for unclosed frontmatter")
	}
}

func TestWithYAMLFrontmatter_EmptyFile(t *testing.T) {
	p := WithYAMLFrontmatter("md", func(payload []byte, meta Envelope) ([]Node, error) {
		return nil, nil
	})

	env, nodes, err := p.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty envelope, got %v", env)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes, got %v", nodes)
	}
}

func TestWithYAMLFrontmatter_ParseFuncError(t *testing.T) {
	p := WithYAMLFrontmatter("md", func(payload []byte, meta Envelope) ([]Node, error) {
		return nil, fmt.Errorf("parse failed")
	})

	raw := []byte("---\ntitle: Test\n---\ncontent")
	_, _, err := p.Parse(raw)
	if err == nil {
		t.Fatal("expected error from parse func")
	}
}

func TestWithNoEnvelope(t *testing.T) {
	p := WithNoEnvelope("dockerfile", func(raw []byte) (Envelope, []Node, error) {
		content := strings.TrimSpace(string(raw))
		env := Envelope{"title": "Dockerfile"}
		nodes := []Node{{Type: "instruction", Content: content}}
		return env, nodes, nil
	})

	if p.Type() != "dockerfile" {
		t.Errorf("expected type 'dockerfile', got %q", p.Type())
	}

	raw := []byte("FROM golang:1.22\nRUN go build")
	env, nodes, err := p.Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if env["title"] != "Dockerfile" {
		t.Errorf("expected title 'Dockerfile', got %v", env["title"])
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
}

func TestWithNoEnvelope_EmptyFile(t *testing.T) {
	p := WithNoEnvelope("raw", func(raw []byte) (Envelope, []Node, error) {
		return Envelope{}, nil, nil
	})

	env, nodes, err := p.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty envelope, got %v", env)
	}
	if len(nodes) != 0 {
		t.Errorf("expected no nodes, got %v", nodes)
	}
}

func TestWrappersImplementParserInterface(t *testing.T) {
	var _ Parser = WithYAMLFrontmatter("a", func([]byte, Envelope) ([]Node, error) { return nil, nil })
	var _ Parser = WithNoEnvelope("b", func([]byte) (Envelope, []Node, error) { return nil, nil, nil })
}
