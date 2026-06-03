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
func (m *mockParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	return core.Envelope{}, []core.Node{{Type: m.typ, Content: string(raw)}}, nil
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

func TestAfterParse(t *testing.T) {
	eng := New()

	called := false
	eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
		called = true
		return pages, nil
	})

	if len(eng.hooks.AfterParse) != 1 {
		t.Fatalf("expected 1 AfterParse hook, got %d", len(eng.hooks.AfterParse))
	}

	// Invoke to verify it works
	_, err := eng.hooks.AfterParse[0](nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("AfterParse hook was not called")
	}
}

func TestBeforeRender(t *testing.T) {
	eng := New()

	eng.BeforeRender(func(pages []*core.Page) ([]*core.Page, error) {
		return pages, nil
	})

	if len(eng.hooks.BeforeRender) != 1 {
		t.Fatalf("expected 1 BeforeRender hook, got %d", len(eng.hooks.BeforeRender))
	}
}

func TestMultipleHooks_FIFO(t *testing.T) {
	eng := New()

	var order []int
	eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
		order = append(order, 1)
		return pages, nil
	})
	eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
		order = append(order, 2)
		return pages, nil
	})

	pages := []*core.Page{{RelPath: "test.md"}}
	for _, hook := range eng.hooks.AfterParse {
		pages, _ = hook(pages)
	}

	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected FIFO order [1,2], got %v", order)
	}
}

func TestAfterParse_FilterPages(t *testing.T) {
	eng := New()

	eng.AfterParse(func(pages []*core.Page) ([]*core.Page, error) {
		var filtered []*core.Page
		for _, p := range pages {
			if p.Envelope["draft"] != true {
				filtered = append(filtered, p)
			}
		}
		return filtered, nil
	})

	pages := []*core.Page{
		{RelPath: "published.md", Envelope: core.Envelope{"title": "Published"}},
		{RelPath: "draft.md", Envelope: core.Envelope{"title": "Draft", "draft": true}},
	}

	result, err := eng.hooks.AfterParse[0](pages)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 page after filtering, got %d", len(result))
	}
	if result[0].RelPath != "published.md" {
		t.Errorf("expected published.md, got %q", result[0].RelPath)
	}
}
