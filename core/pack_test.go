package core

import "testing"

type stubParser struct{ typ string }

func (p *stubParser) Type() string { return p.typ }
func (p *stubParser) Parse(raw []byte) (Envelope, []Node, error) {
	return Envelope{}, nil, nil
}

func TestPackContextConfigAndRegistration(t *testing.T) {
	cfg := map[string]any{"enabled": true, "level": 3}
	pc := NewPackContext("cards", cfg)

	if pc.Name() != "cards" {
		t.Errorf("Name = %q", pc.Name())
	}
	if pc.Config()["level"] != 3 {
		t.Errorf("Config not exposed: %+v", pc.Config())
	}

	pc.Register(&stubParser{typ: "card"})
	noop := func(pages []*Page) ([]*Page, error) { return pages, nil }
	pc.AfterParse(noop)
	pc.Index(noop)
	pc.BeforeRender(noop)

	parsers, hooks := pc.Registered()
	if len(parsers) != 1 || parsers[0].Type() != "card" {
		t.Errorf("registered parsers = %+v", parsers)
	}
	if len(hooks.AfterParse) != 1 || len(hooks.Index) != 1 || len(hooks.BeforeRender) != 1 {
		t.Errorf("registered hooks not captured: %+v", hooks)
	}
}

func TestPackContextNilConfig(t *testing.T) {
	pc := NewPackContext("x", nil)
	if pc.Config() != nil {
		t.Error("nil config should stay nil")
	}
	parsers, hooks := pc.Registered()
	if parsers != nil || len(hooks.AfterParse) != 0 {
		t.Error("fresh context should have no registrations")
	}
}
