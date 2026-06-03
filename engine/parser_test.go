package engine

import (
	"testing"

	"github.com/FabioSol/fuego/core"
)

func TestNodeJSONTags(t *testing.T) {
	n := core.Node{
		Type:       "question",
		Attributes: map[string]any{"correct": true},
		Content:    "What is Go?",
		Children: []core.Node{
			{Type: "option", Content: "A language", Attributes: map[string]any{"correct": true}},
			{Type: "option", Content: "A framework", Attributes: map[string]any{"correct": false}},
		},
	}

	if n.Type != "question" {
		t.Errorf("expected type 'question', got %q", n.Type)
	}
	if len(n.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(n.Children))
	}
	if n.Children[0].Attributes["correct"] != true {
		t.Error("first child should be correct")
	}
}

func TestParserInterface(t *testing.T) {
	var p core.Parser = &mockParser{typ: "test"}

	if p.Type() != "test" {
		t.Errorf("expected type 'test', got %q", p.Type())
	}

	_, nodes, err := p.Parse([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Content != "hello" {
		t.Errorf("expected content 'hello', got %q", nodes[0].Content)
	}
}
