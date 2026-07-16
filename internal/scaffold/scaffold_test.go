package scaffold_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
	"github.com/gofuego/fuego/internal/formats"
	"github.com/gofuego/fuego/internal/pipeline"
	"github.com/gofuego/fuego/internal/scaffold"
	"github.com/gofuego/fuego/parsers/markdown"
)

// defaultFormats is what `fuego init` without --formats resolves to.
func defaultFormats(t *testing.T) []formats.Format {
	t.Helper()
	md, err := formats.Resolve("markdown")
	if err != nil {
		t.Fatal(err)
	}
	return []formats.Format{md}
}

// TestScaffoldBuilds generates a project and builds it through the real
// pipeline (offline — no go get), proving the scaffold a new user receives
// renders cleanly and demonstrates the v0.3 surface.
func TestScaffoldBuilds(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "Demo Site", Module: "demo", Formats: defaultFormats(t)}); err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	cfg, err := config.Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("loading scaffold config: %v", err)
	}
	cfg.Dirs.Content = filepath.Join(dir, cfg.Dirs.Content)
	cfg.Dirs.Theme = filepath.Join(dir, cfg.Dirs.Theme)
	cfg.Dirs.Static = filepath.Join(dir, cfg.Dirs.Static)
	cfg.Dirs.Output = filepath.Join(dir, "build")

	// The scaffold's formats.go registers Markdown; the card parser is
	// declarative. (The generated program isn't compiled here — the in-process
	// build registers the same parser set formats.go would.)
	parsers := map[string]core.Parser{"md": markdown.Parser()}

	if err := pipeline.Build(context.Background(), cfg, parsers, nil, nil, pipeline.Options{}); err != nil {
		t.Fatalf("scaffold site failed to build: %v", err)
	}

	out := cfg.Dirs.Output
	mustExist := []string{
		"index.html",              // Markdown home page (proves md parser wired)
		"cards/index.html",        // paginated collection, page 1
		"cards/page/2/index.html", // pagination
		"sitemap.xml",             // theme/outputs/
		"rss.xml",                 // theme/outputs/
		"site-manifest.json",
	}
	for _, rel := range mustExist {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("expected build output %s: %v", rel, err)
		}
	}

	// The home page must come from the Markdown parser, not be a copied asset.
	home, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "Your Fuego site is running") {
		t.Error("home page did not render Markdown content")
	}
	// Nav is partial-driven off .Site.Pages.
	if !strings.Contains(string(home), "/cards/") {
		t.Error("nav partial did not render the cards link")
	}
}

// TestScaffoldFormatsFile locks the tool-owned registration file: main.go
// calls registerFormats, formats.go imports and registers the exact --formats
// set, and the CLAUDE.md points at the docs area.
func TestScaffoldFormatsFile(t *testing.T) {
	md, _ := formats.Resolve("markdown")
	mermaid, _ := formats.Resolve("mermaid")
	openapi, _ := formats.Resolve("openapi")

	dir := t.TempDir()
	err := scaffold.WriteFiles(dir, scaffold.Data{
		Name: "demo", Module: "demo",
		Formats: []formats.Format{md, mermaid, openapi},
	})
	if err != nil {
		t.Fatal(err)
	}

	main, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if !strings.Contains(string(main), "registerFormats(eng)") {
		t.Errorf("main.go must call registerFormats:\n%s", main)
	}
	if strings.Contains(string(main), "parsers/markdown") {
		t.Errorf("main.go must not register formats inline anymore:\n%s", main)
	}

	ff, err := os.ReadFile(filepath.Join(dir, formats.FileName))
	if err != nil {
		t.Fatalf("formats.go missing: %v", err)
	}
	for _, want := range []string{
		`"github.com/gofuego/fuego/parsers/markdown"`,
		`"github.com/gofuego/fuego-formats/mermaid"`,
		`"github.com/gofuego/fuego-formats/openapi"`,
		"eng.Register(markdown.Parser())",
		"eng.Register(mermaid.Parser())",
		"eng.Register(openapi.Parser())",
	} {
		if !strings.Contains(string(ff), want) {
			t.Errorf("formats.go missing %q:\n%s", want, ff)
		}
	}

	claude, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if !strings.Contains(string(claude), "docs/formats/README.md") {
		t.Error("CLAUDE.md must point at the materialized format docs index")
	}
}

func TestScaffoldDeterministic(t *testing.T) {
	read := func() map[string]string {
		dir := t.TempDir()
		if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "X", Module: "x", Formats: defaultFormats(t)}); err != nil {
			t.Fatal(err)
		}
		files := map[string]string{}
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(dir, p)
			b, _ := os.ReadFile(p)
			files[rel] = string(b)
			return nil
		})
		return files
	}

	a, b := read(), read()
	if len(a) != len(b) {
		t.Fatalf("file count differs: %d vs %d", len(a), len(b))
	}
	for k, va := range a {
		if b[k] != va {
			t.Errorf("file %s differs between generations", k)
		}
	}
}
