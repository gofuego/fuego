package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestGenerate_BasicPages(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/b/", Type: "md", Envelope: core.Envelope{"title": "B"}},
		{URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
	}

	cfg := &config.Config{}
	m := Generate(pages, cfg)

	if len(m.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(m.Pages))
	}

	// Sorted by URL
	if m.Pages[0].URL != "/a/" {
		t.Errorf("expected first page /a/, got %q", m.Pages[0].URL)
	}
	if m.Pages[1].URL != "/b/" {
		t.Errorf("expected second page /b/, got %q", m.Pages[1].URL)
	}
	if m.Pages[0].Title != "A" {
		t.Errorf("expected title 'A', got %q", m.Pages[0].Title)
	}
}

func TestGenerate_WithSummary(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "summary": "A short summary",
		}},
	}

	cfg := &config.Config{}
	m := Generate(pages, cfg)

	if m.Pages[0].Summary != "A short summary" {
		t.Errorf("expected summary, got %q", m.Pages[0].Summary)
	}
}

func TestGenerate_TaxonomyIntegerIndexes(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "tags": []any{"go", "web"},
		}},
		{URL: "/b/", Type: "md", Envelope: core.Envelope{
			"title": "B", "tags": []any{"go"},
		}},
		// Virtual taxonomy pages should be excluded from term indexes
		{URL: "/tags/go/", Type: "taxonomy-term", Envelope: core.Envelope{
			"taxonomy": "tags", "term": "go",
		}},
	}

	cfg := &config.Config{
		Taxonomies: map[string]config.TaxonomyConfig{
			"tags": {Path: "/tags/{term}", IndexPath: "/tags"},
		},
	}

	m := Generate(pages, cfg)

	if m.Taxonomies == nil {
		t.Fatal("expected taxonomies in manifest")
	}

	tags, ok := m.Taxonomies["tags"]
	if !ok {
		t.Fatal("expected 'tags' taxonomy")
	}

	// /a/ is at index 0, /b/ is at index 1 (sorted by URL)
	goIndexes := tags.Terms["go"]
	if len(goIndexes) != 2 {
		t.Fatalf("expected 2 pages for 'go', got %d", len(goIndexes))
	}
	if goIndexes[0] != 0 || goIndexes[1] != 1 {
		t.Errorf("expected [0,1] for 'go', got %v", goIndexes)
	}

	webIndexes := tags.Terms["web"]
	if len(webIndexes) != 1 || webIndexes[0] != 0 {
		t.Errorf("expected [0] for 'web', got %v", webIndexes)
	}
}

func TestGenerate_CollectionIntegerIndexes(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/history/q1/", Type: "trivia", RelPath: "history/q1.trivia", Envelope: core.Envelope{
			"title": "Q1", "points": 30,
		}},
		{URL: "/history/q2/", Type: "trivia", RelPath: "history/q2.trivia", Envelope: core.Envelope{
			"title": "Q2", "points": 10,
		}},
		// Virtual collection page with page-ref nodes (sorted by points: q2 first)
		{URL: "/history-trivia/", Type: "collection", Envelope: core.Envelope{
			"collection": "history-trivia",
		}, Nodes: []core.Node{
			{Type: "page-ref", Attributes: map[string]any{"url": "/history/q2/"}},
			{Type: "page-ref", Attributes: map[string]any{"url": "/history/q1/"}},
		}},
	}

	cfg := &config.Config{
		Collections: map[string]config.CollectionConfig{
			"history-trivia": {Match: "history/**", SortBy: "points", Path: "/history-trivia"},
		},
	}

	m := Generate(pages, cfg)

	if m.Collections == nil {
		t.Fatal("expected collections in manifest")
	}

	col, ok := m.Collections["history-trivia"]
	if !ok {
		t.Fatal("expected 'history-trivia' collection")
	}

	// Pages sorted by URL: /history-trivia/ (idx 0), /history/q1/ (idx 1), /history/q2/ (idx 2)
	// Collection references q2 first (points=10), then q1 (points=30)
	if len(col.Pages) != 2 {
		t.Fatalf("expected 2 pages in collection, got %d", len(col.Pages))
	}
	if col.Pages[0] != 2 { // /history/q2/
		t.Errorf("expected first collection member index 2, got %d", col.Pages[0])
	}
	if col.Pages[1] != 1 { // /history/q1/
		t.Errorf("expected second collection member index 1, got %d", col.Pages[1])
	}
}

func TestGenerate_DeterministicJSON(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/z/", Type: "md", Envelope: core.Envelope{"title": "Z"}},
		{URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
	}

	cfg := &config.Config{}

	m1 := Generate(pages, cfg)
	m2 := Generate(pages, cfg)

	j1, _ := json.Marshal(m1)
	j2, _ := json.Marshal(m2)

	if string(j1) != string(j2) {
		t.Error("manifest is not deterministic")
	}
}

func TestGenerate_NoTaxonomiesOrCollections(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
	}

	cfg := &config.Config{}
	m := Generate(pages, cfg)

	if m.Taxonomies != nil {
		t.Error("expected nil taxonomies when not configured")
	}
	if m.Collections != nil {
		t.Error("expected nil collections when not configured")
	}
}

func TestWrite_CreatesFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	m := &Manifest{
		Pages: []PageEntry{
			{URL: "/a/", Type: "md", Title: "A"},
		},
	}

	if err := Write(m, dir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "site-manifest.json"))
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}

	var loaded Manifest
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("parsing manifest: %v", err)
	}

	if len(loaded.Pages) != 1 || loaded.Pages[0].URL != "/a/" {
		t.Errorf("unexpected manifest content: %s", data)
	}
}
