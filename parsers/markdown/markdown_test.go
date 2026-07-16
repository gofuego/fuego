package markdown

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
)

var update = flag.Bool("update", false, "regenerate golden fixtures")

func TestParser_Type(t *testing.T) {
	p := Parser()
	if p.Type() != "md" {
		t.Errorf("expected type 'md', got %q", p.Type())
	}
}

// The default claim is the bare md extension: Parser() must NOT implement
// core.FilenameParser, or the resolver would treat its claim as a filename
// pattern and change its rank under specificity dispatch.
func TestParser_DefaultClaimIsBareExtension(t *testing.T) {
	if _, ok := Parser().(core.FilenameParser); ok {
		t.Error("Parser() without options must not implement core.FilenameParser")
	}
}

func TestParser_WithPatternsOverridesClaims(t *testing.T) {
	fp, ok := Parser(WithPatterns("README.md", "*.markdown")).(core.FilenameParser)
	if !ok {
		t.Fatal("Parser(WithPatterns(...)) must implement core.FilenameParser")
	}
	got := fp.Filenames()
	if len(got) != 2 || got[0] != "README.md" || got[1] != "*.markdown" {
		t.Errorf("Filenames() = %v, want [README.md *.markdown]", got)
	}
	if fp.Type() != Type {
		t.Errorf("Type() = %q, want %q — overriding claims must not change the page type", fp.Type(), Type)
	}
}

func TestParser_WithPatternsStillParses(t *testing.T) {
	p := Parser(WithPatterns("README.md"))
	env, nodes, err := p.Parse([]byte("---\ntitle: Readme\n---\nHello."))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["title"] != "Readme" {
		t.Errorf("title = %v, want Readme", env["title"])
	}
	if len(nodes) != 1 || nodes[0].Type != NodeHTML {
		t.Fatalf("expected one %q node, got %v", NodeHTML, nodes)
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

// dump is the golden node-dump: the parser's full output for a fixture input,
// serialized deterministically — the same shape fuego-formats modules ship as
// their contract example.
type dump struct {
	Envelope core.Envelope `json:"envelope"`
	Nodes    []core.Node   `json:"nodes"`
}

// TestGoldenDump is simultaneously the regression test and the shipped
// contract example. Regenerate with: go test ./parsers/markdown -update
func TestGoldenDump(t *testing.T) {
	inputs, err := filepath.Glob(filepath.Join("testdata", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(inputs) == 0 {
		t.Fatal("no testdata/*.md fixtures found")
	}

	for _, in := range inputs {
		name := strings.TrimSuffix(filepath.Base(in), ".md")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			raw, err := os.ReadFile(in)
			if err != nil {
				t.Fatal(err)
			}
			env, nodes, err := Parser().Parse(raw)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(dump{Envelope: env, Nodes: nodes}, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, '\n')

			golden := filepath.Join("testdata", name+".golden.json")
			if *update {
				if err := os.WriteFile(golden, got, 0644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("reading golden (run with -update): %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
			}
		})
	}
}

// TestSchemaSections mirrors the fuego-formats schemalint: schema.md is this
// parser's contract and must keep the six required sections.
func TestSchemaSections(t *testing.T) {
	raw, err := os.ReadFile("schema.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range []string{
		"## Claims",
		"## Envelope keys",
		"## Node types",
		"## Tree shape",
		"## Slug derivation",
		"## Stability",
	} {
		if !strings.Contains(string(raw), section+"\n") {
			t.Errorf("schema.md missing required section %q", section)
		}
	}
}
