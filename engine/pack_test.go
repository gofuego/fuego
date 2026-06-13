package engine

import (
	"testing"

	"github.com/FabioSol/fuego/core"
)

type fakeParser struct{ typ string }

func (p *fakeParser) Type() string { return p.typ }
func (p *fakeParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	return core.Envelope{}, nil, nil
}

func TestUseRegistersPackParsers(t *testing.T) {
	eng := New()
	packParser := &fakeParser{typ: "card"}
	eng.Use(core.Pack{Name: "cards", Parsers: []core.Parser{packParser}})

	if eng.Parsers()["card"] != packParser {
		t.Error("pack parser not registered")
	}
}

func TestUserParserWinsOverPack(t *testing.T) {
	userParser := &fakeParser{typ: "card"}
	packParser := &fakeParser{typ: "card"}

	// Register before Use
	eng := New()
	eng.Register(userParser)
	eng.Use(core.Pack{Name: "cards", Parsers: []core.Parser{packParser}})
	if eng.Parsers()["card"] != userParser {
		t.Error("user parser registered before Use should win")
	}

	// Register after Use
	eng = New()
	eng.Use(core.Pack{Name: "cards", Parsers: []core.Parser{packParser}})
	eng.Register(userParser)
	if eng.Parsers()["card"] != userParser {
		t.Error("user parser registered after Use should win")
	}
}

func TestLaterPackWinsOverEarlier(t *testing.T) {
	first := &fakeParser{typ: "card"}
	second := &fakeParser{typ: "card"}

	eng := New()
	eng.Use(core.Pack{Name: "first", Parsers: []core.Parser{first}})
	eng.Use(core.Pack{Name: "second", Parsers: []core.Parser{second}})

	if eng.Parsers()["card"] != second {
		t.Error("later pack should override earlier pack's parser")
	}
}

func TestUseAppendsHooks(t *testing.T) {
	eng := New()
	noop := func(pages []*core.Page) ([]*core.Page, error) { return pages, nil }

	eng.AfterParse(noop)
	eng.Use(core.Pack{
		Name: "cards",
		Hooks: core.Hooks{
			AfterParse:   []core.AfterParseHook{noop},
			Index:        []core.IndexHook{noop},
			BeforeRender: []core.BeforeRenderHook{noop},
		},
	})

	if len(eng.hooks.AfterParse) != 2 {
		t.Errorf("AfterParse hooks = %d, want 2 (user + pack)", len(eng.hooks.AfterParse))
	}
	if len(eng.hooks.Index) != 1 || len(eng.hooks.BeforeRender) != 1 {
		t.Error("pack Index/BeforeRender hooks not appended")
	}
}
