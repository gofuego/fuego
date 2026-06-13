package scaffold_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/FabioSol/fuego/internal/scaffold"
	"github.com/FabioSol/fuego/parsers/markdown"
)

// TestScaffoldBuilds generates a project and builds it through the real
// pipeline (offline — no go get), proving the scaffold a new user receives
// renders cleanly and demonstrates the v0.3 surface.
func TestScaffoldBuilds(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "Demo Site", Module: "demo"}); err != nil {
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

	// The scaffold's main.go registers Markdown; the card parser is declarative.
	parsers := map[string]core.Parser{"md": markdown.Parser()}

	if err := pipeline.Build(context.Background(), cfg, parsers, nil, nil); err != nil {
		t.Fatalf("scaffold site failed to build: %v", err)
	}

	out := cfg.Dirs.Output
	mustExist := []string{
		"index/index.html",        // Markdown home page (proves md parser wired)
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
	home, err := os.ReadFile(filepath.Join(out, "index", "index.html"))
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

func TestScaffoldDeterministic(t *testing.T) {
	read := func() map[string]string {
		dir := t.TempDir()
		if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "X", Module: "x"}); err != nil {
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
