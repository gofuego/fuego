package parse

import (
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
)

func TestNewDeclarativeParser_InvalidRegex(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: "[invalid", Emit: config.EmitConfig{Type: "x"}},
		},
	}
	_, err := NewDeclarativeParser("bad", cfg)
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestNewDeclarativeParser_ValidRegex(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `^front:\s*(.+)$`, Emit: config.EmitConfig{Type: "front", Content: "$1"}},
		},
	}
	dp, err := NewDeclarativeParser("card", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dp.Type() != "card" {
		t.Errorf("expected type 'card', got %q", dp.Type())
	}
}

func TestDeclarativeParser_SingleRule(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `^front:\s*(.+)$`, Emit: config.EmitConfig{Type: "front", Content: "$1"}},
		},
	}
	dp, _ := NewDeclarativeParser("card", cfg)

	_, nodes, err := dp.Parse([]byte("front: What is Go?"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Type != "front" {
		t.Errorf("expected type 'front', got %q", nodes[0].Type)
	}
	if nodes[0].Content != "What is Go?" {
		t.Errorf("expected content 'What is Go?', got %q", nodes[0].Content)
	}
}

func TestDeclarativeParser_MultiRule_FirstMatchWins(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `^front:\s*(.+)$`, Emit: config.EmitConfig{Type: "front", Content: "$1"}},
			{Match: `^back:\s*(.+)$`, Emit: config.EmitConfig{Type: "back", Content: "$1"}},
		},
	}
	dp, _ := NewDeclarativeParser("card", cfg)

	_, nodes, err := dp.Parse([]byte("front: Q\nback: A"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].Type != "front" || nodes[0].Content != "Q" {
		t.Errorf("node 0: got type=%q content=%q", nodes[0].Type, nodes[0].Content)
	}
	if nodes[1].Type != "back" || nodes[1].Content != "A" {
		t.Errorf("node 1: got type=%q content=%q", nodes[1].Type, nodes[1].Content)
	}
}

func TestDeclarativeParser_CaptureGroups(t *testing.T) {
	t.Parallel()
	// Match: "[X] Answer text" → emit type=answer, content=Answer text, attributes.correct=$1
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{
				Match: `^\[([A-Z])\]\s+(.+)$`,
				Emit: config.EmitConfig{
					Type:    "answer",
					Content: "$2",
					Attributes: map[string]any{
						"letter": "$1",
						"full":   "$0",
					},
				},
			},
		},
	}
	dp, _ := NewDeclarativeParser("trivia", cfg)

	_, nodes, err := dp.Parse([]byte("[A] Paris is the capital"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	n := nodes[0]
	if n.Type != "answer" {
		t.Errorf("expected type 'answer', got %q", n.Type)
	}
	if n.Content != "Paris is the capital" {
		t.Errorf("expected content 'Paris is the capital', got %q", n.Content)
	}
	if n.Attributes["letter"] != "A" {
		t.Errorf("expected letter 'A', got %v", n.Attributes["letter"])
	}
	if n.Attributes["full"] != "[A] Paris is the capital" {
		t.Errorf("expected full match, got %v", n.Attributes["full"])
	}
}

func TestDeclarativeParser_UnmatchedLinesSkipped(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `^front:\s*(.+)$`, Emit: config.EmitConfig{Type: "front", Content: "$1"}},
		},
	}
	dp, _ := NewDeclarativeParser("card", cfg)

	_, nodes, err := dp.Parse([]byte("front: Q\nthis line matches nothing\nfront: Q2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (unmatched skipped), got %d", len(nodes))
	}
}

func TestDeclarativeParser_EmptyPayload(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `^front:\s*(.+)$`, Emit: config.EmitConfig{Type: "front", Content: "$1"}},
		},
	}
	dp, _ := NewDeclarativeParser("card", cfg)

	_, nodes, err := dp.Parse([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes for empty payload, got %d", len(nodes))
	}
}

func TestDeclarativeParser_WholeLineMatch(t *testing.T) {
	t.Parallel()
	// $0 captures the entire match
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `.+`, Emit: config.EmitConfig{Type: "line", Content: "$0"}},
		},
	}
	dp, _ := NewDeclarativeParser("raw", cfg)

	_, nodes, err := dp.Parse([]byte("hello world"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Content != "hello world" {
		t.Errorf("expected 'hello world', got %q", nodes[0].Content)
	}
}

func TestDeclarativeParser_NonStringAttributesPassedThrough(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{
				Match: `.+`,
				Emit: config.EmitConfig{
					Type:    "item",
					Content: "$0",
					Attributes: map[string]any{
						"weight": 42,
						"active": true,
					},
				},
			},
		},
	}
	dp, _ := NewDeclarativeParser("test", cfg)

	_, nodes, err := dp.Parse([]byte("some line"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nodes[0].Attributes["weight"] != 42 {
		t.Errorf("expected weight 42, got %v", nodes[0].Attributes["weight"])
	}
	if nodes[0].Attributes["active"] != true {
		t.Errorf("expected active true, got %v", nodes[0].Attributes["active"])
	}
}

func TestDeclarativeParser_ImplementsParserInterface(t *testing.T) {
	t.Parallel()
	cfg := config.ParserConfig{
		Rules: []config.RuleConfig{
			{Match: `.+`, Emit: config.EmitConfig{Type: "line", Content: "$0"}},
		},
	}
	dp, _ := NewDeclarativeParser("test", cfg)

	// Verify it satisfies the core.Parser interface
	var _ core.Parser = dp
}
