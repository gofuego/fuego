package dispatch

import (
	"testing"

	"github.com/gofuego/fuego/core"
)

// extParser claims only a bare extension (its Type).
type extParser struct{ typ string }

func (p extParser) Type() string                                     { return p.typ }
func (p extParser) Parse([]byte) (core.Envelope, []core.Node, error) { return nil, nil, nil }

// patParser claims one or more filename patterns; patterns are its complete
// claim set (its Type is not an extension claim).
type patParser struct {
	typ      string
	patterns []string
}

func (p patParser) Type() string                                     { return p.typ }
func (p patParser) Parse([]byte) (core.Envelope, []core.Node, error) { return nil, nil, nil }
func (p patParser) Filenames() []string                              { return p.patterns }

func TestResolveExtensionOnly(t *testing.T) {
	r := NewResolver([]core.Parser{extParser{typ: "md"}})

	got, ok := r.Resolve("guide.md")
	if !ok || got != "md" {
		t.Fatalf("guide.md: got (%q,%v), want (\"md\",true)", got, ok)
	}
}

func TestResolveNoMatchIsAsset(t *testing.T) {
	r := NewResolver([]core.Parser{extParser{typ: "md"}})

	if got, ok := r.Resolve("photo.png"); ok {
		t.Errorf("photo.png should be an asset, got parser %q", got)
	}
	if got, ok := r.Resolve("noext"); ok {
		t.Errorf("extensionless unclaimed file should be an asset, got parser %q", got)
	}
}

// A filename pattern must beat a bare-extension claim on the same file, even
// though the extension ("md") is also registered.
func TestResolvePatternBeatsExtension(t *testing.T) {
	r := NewResolver([]core.Parser{
		extParser{typ: "md"},
		patParser{typ: "adr", patterns: []string{"*.adr.md"}},
	})

	if got, ok := r.Resolve("0001-decision.adr.md"); !ok || got != "adr" {
		t.Errorf("*.adr.md file: got (%q,%v), want (\"adr\",true)", got, ok)
	}
	// A plain markdown file still routes to the extension parser.
	if got, ok := r.Resolve("readme.md"); !ok || got != "md" {
		t.Errorf("readme.md: got (%q,%v), want (\"md\",true)", got, ok)
	}
}

// When several patterns match, the longest pattern string wins.
func TestResolveLongestPatternWins(t *testing.T) {
	r := NewResolver([]core.Parser{
		patParser{typ: "wild", patterns: []string{"*"}},
		patParser{typ: "md-ish", patterns: []string{"*.md"}},
		patParser{typ: "adr", patterns: []string{"*.adr.md"}},
	})

	if got, ok := r.Resolve("x.adr.md"); !ok || got != "adr" {
		t.Errorf("x.adr.md: got (%q,%v), want (\"adr\",true) — longest pattern *.adr.md", got, ok)
	}
	if got, ok := r.Resolve("x.md"); !ok || got != "md-ish" {
		t.Errorf("x.md: got (%q,%v), want (\"md-ish\",true) — *.md beats *", got, ok)
	}
	if got, ok := r.Resolve("plainfile"); !ok || got != "wild" {
		t.Errorf("plainfile: got (%q,%v), want (\"wild\",true) — only * matches", got, ok)
	}
}

// Equal-length patterns matching the same file tie; the higher-precedence
// (later-supplied) parser wins. NewResolver takes parsers lowest-precedence
// first.
func TestResolveTieBreaksByPrecedence(t *testing.T) {
	// Two parsers claim identical-length patterns that both match the target,
	// forcing a genuine tie resolved by precedence.
	a := patParser{typ: "a", patterns: []string{"*.ab.md"}}
	b := patParser{typ: "b", patterns: []string{"*.ab.md"}}

	// a supplied before b → b has higher precedence and wins the tie.
	r := NewResolver([]core.Parser{a, b})
	if got, ok := r.Resolve("file.ab.md"); !ok || got != "b" {
		t.Errorf("tie file.ab.md: got (%q,%v), want (\"b\",true) — later parser wins tie", got, ok)
	}

	// Reverse the order: now a wins.
	r2 := NewResolver([]core.Parser{b, a})
	if got, ok := r2.Resolve("file.ab.md"); !ok || got != "a" {
		t.Errorf("reversed tie file.ab.md: got (%q,%v), want (\"a\",true)", got, ok)
	}
}

// Extensionless files (Dockerfile) are claimed by a literal filename pattern,
// never by an extension (they have none).
func TestResolveExtensionlessFilename(t *testing.T) {
	r := NewResolver([]core.Parser{
		extParser{typ: "md"},
		patParser{typ: "dockerfile", patterns: []string{"Dockerfile"}},
	})

	if got, ok := r.Resolve("Dockerfile"); !ok || got != "dockerfile" {
		t.Errorf("Dockerfile: got (%q,%v), want (\"dockerfile\",true)", got, ok)
	}
	// A wildcard variant still matches when declared.
	r2 := NewResolver([]core.Parser{
		patParser{typ: "dockerfile", patterns: []string{"Dockerfile*"}},
	})
	if got, ok := r2.Resolve("Dockerfile.prod"); !ok || got != "dockerfile" {
		t.Errorf("Dockerfile.prod: got (%q,%v), want (\"dockerfile\",true)", got, ok)
	}
	if _, ok := r2.Resolve("Makefile"); ok {
		t.Error("Makefile should not match Dockerfile* pattern")
	}
}

// Declared patterns are a parser's complete claim set: a pattern parser's
// Type() is not implicitly claimed as an extension. This is what makes claim
// overrides total — a markdown parser re-claimed as README.md must not keep
// claiming every .md file through its "md" type.
func TestResolvePatternsAreCompleteClaimSet(t *testing.T) {
	r := NewResolver([]core.Parser{
		patParser{typ: "md", patterns: []string{"README.md"}},
	})

	if got, ok := r.Resolve("README.md"); !ok || got != "md" {
		t.Errorf("README.md: got (%q,%v), want (\"md\",true)", got, ok)
	}
	if got, ok := r.Resolve("guide.md"); ok {
		t.Errorf("guide.md should be an asset when the md parser claims only README.md, got parser %q", got)
	}
}

// A FilenameParser reporting no patterns degrades to a plain extension claim
// rather than claiming nothing.
func TestResolveEmptyPatternsFallBackToExtension(t *testing.T) {
	r := NewResolver([]core.Parser{patParser{typ: "md"}})

	if got, ok := r.Resolve("guide.md"); !ok || got != "md" {
		t.Errorf("guide.md: got (%q,%v), want (\"md\",true)", got, ok)
	}
}

// Type() is the parser's own type regardless of whether it matched by pattern
// or extension: the resolver returns the parser type, and PARSE sets page.Type
// to it (Ext stays the literal extension, tested at the integration layer).
func TestResolveReturnsParserType(t *testing.T) {
	r := NewResolver([]core.Parser{
		extParser{typ: "md"},
		patParser{typ: "adr", patterns: []string{"*.adr.md"}},
	})

	// Both the extension match and the pattern match return the parser's Type().
	if got, _ := r.Resolve("plain.md"); got != "md" {
		t.Errorf("plain.md parser type = %q, want md", got)
	}
	if got, _ := r.Resolve("d.adr.md"); got != "adr" {
		t.Errorf("d.adr.md parser type = %q, want adr", got)
	}
}
