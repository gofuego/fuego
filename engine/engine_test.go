package engine

import (
	"testing"

	"github.com/FabioSol/fuego/core"
)

// mockParser is a minimal Parser implementation for testing.
type mockParser struct {
	typ string
}

func (m *mockParser) Type() string { return m.typ }
func (m *mockParser) Parse(raw []byte, meta core.Envelope) ([]core.Node, error) {
	return []core.Node{{Type: m.typ, Content: string(raw)}}, nil
}

func TestNew(t *testing.T) {
	eng := New()
	if eng == nil {
		t.Fatal("New() returned nil")
	}
	if eng.parsers == nil {
		t.Fatal("parsers map is nil")
	}
	if len(eng.parsers) != 0 {
		t.Fatalf("expected 0 parsers, got %d", len(eng.parsers))
	}
}

func TestRegister(t *testing.T) {
	eng := New()

	eng.Register(&mockParser{typ: "trivia"})
	eng.Register(&mockParser{typ: "chess"})

	if len(eng.parsers) != 2 {
		t.Fatalf("expected 2 parsers, got %d", len(eng.parsers))
	}
	if _, ok := eng.parsers["trivia"]; !ok {
		t.Error("trivia parser not found")
	}
	if _, ok := eng.parsers["chess"]; !ok {
		t.Error("chess parser not found")
	}
}

func TestRegisterOverwrite(t *testing.T) {
	eng := New()

	p1 := &mockParser{typ: "trivia"}
	p2 := &mockParser{typ: "trivia"}

	eng.Register(p1)
	eng.Register(p2)

	if len(eng.parsers) != 1 {
		t.Fatalf("expected 1 parser after overwrite, got %d", len(eng.parsers))
	}
}

func TestParsers(t *testing.T) {
	eng := New()
	eng.Register(&mockParser{typ: "card"})

	p := eng.Parsers()
	if len(p) != 1 {
		t.Fatalf("expected 1 parser, got %d", len(p))
	}
	if _, ok := p["card"]; !ok {
		t.Error("card parser not found in Parsers() map")
	}
}
