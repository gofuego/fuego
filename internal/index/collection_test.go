package index

import (
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestBuildCollections_GlobMatch(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "history/q1.trivia", URL: "/history/q1/", Type: "trivia", Envelope: core.Envelope{
			"title": "Question 1", "points": 10,
		}},
		{RelPath: "history/q2.trivia", URL: "/history/q2/", Type: "trivia", Envelope: core.Envelope{
			"title": "Question 2", "points": 20,
		}},
		{RelPath: "science/q1.trivia", URL: "/science/q1/", Type: "trivia", Envelope: core.Envelope{
			"title": "Science Q1", "points": 15,
		}},
	}

	colCfg := map[string]config.CollectionConfig{
		"history-trivia": {
			Match:  "history/**",
			SortBy: "points",
			Layout: "listing",
			Path:   "/history-trivia",
		},
	}

	virtual := BuildCollections(pages, colCfg)
	if len(virtual) != 1 {
		t.Fatalf("expected 1 collection page, got %d", len(virtual))
	}

	col := virtual[0]
	if col.URL != "/history-trivia/" {
		t.Errorf("expected /history-trivia/, got %q", col.URL)
	}
	if col.Layout != "listing" {
		t.Errorf("expected layout 'listing', got %q", col.Layout)
	}
	if col.Type != "collection" {
		t.Errorf("expected type 'collection', got %q", col.Type)
	}

	// Should have 2 members (only history/**)
	if len(col.Nodes) != 2 {
		t.Fatalf("expected 2 page-ref nodes, got %d", len(col.Nodes))
	}

	// Sorted by points: 10, 20
	if col.Nodes[0].Attributes["title"] != "Question 1" {
		t.Errorf("expected first node title 'Question 1', got %v", col.Nodes[0].Attributes["title"])
	}
	if col.Nodes[1].Attributes["title"] != "Question 2" {
		t.Errorf("expected second node title 'Question 2', got %v", col.Nodes[1].Attributes["title"])
	}
}

func TestBuildCollections_SortByNumeric(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.trivia", URL: "/a/", Type: "trivia", Envelope: core.Envelope{
			"title": "High", "points": 30,
		}},
		{RelPath: "b.trivia", URL: "/b/", Type: "trivia", Envelope: core.Envelope{
			"title": "Low", "points": 5,
		}},
		{RelPath: "c.trivia", URL: "/c/", Type: "trivia", Envelope: core.Envelope{
			"title": "Mid", "points": 15,
		}},
	}

	colCfg := map[string]config.CollectionConfig{
		"all": {Match: "**/*.trivia", SortBy: "points", Path: "/all"},
	}

	virtual := BuildCollections(pages, colCfg)
	if len(virtual) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(virtual))
	}

	nodes := virtual[0].Nodes
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	// Sorted ascending: 5, 15, 30
	titles := []string{
		nodes[0].Attributes["title"].(string),
		nodes[1].Attributes["title"].(string),
		nodes[2].Attributes["title"].(string),
	}
	expected := []string{"Low", "Mid", "High"}
	for i, want := range expected {
		if titles[i] != want {
			t.Errorf("index %d: got %q, want %q", i, titles[i], want)
		}
	}
}

func TestBuildCollections_SortByString(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "c.md", URL: "/c/", Type: "md", Envelope: core.Envelope{
			"title": "Zulu",
		}},
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "Alpha",
		}},
		{RelPath: "b.md", URL: "/b/", Type: "md", Envelope: core.Envelope{
			"title": "Bravo",
		}},
	}

	colCfg := map[string]config.CollectionConfig{
		"all": {Match: "**/*.md", SortBy: "title", Path: "/all"},
	}

	virtual := BuildCollections(pages, colCfg)
	nodes := virtual[0].Nodes

	titles := []string{
		nodes[0].Attributes["title"].(string),
		nodes[1].Attributes["title"].(string),
		nodes[2].Attributes["title"].(string),
	}
	expected := []string{"Alpha", "Bravo", "Zulu"}
	for i, want := range expected {
		if titles[i] != want {
			t.Errorf("index %d: got %q, want %q", i, titles[i], want)
		}
	}
}

func TestBuildCollections_NoMatches(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
	}

	colCfg := map[string]config.CollectionConfig{
		"empty": {Match: "**/*.trivia", Path: "/empty"},
	}

	virtual := BuildCollections(pages, colCfg)
	if len(virtual) != 0 {
		t.Errorf("expected 0 virtual pages for no matches, got %d", len(virtual))
	}
}

func TestBuildCollections_Envelope(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
	}

	colCfg := map[string]config.CollectionConfig{
		"docs": {Match: "**/*.md", Path: "/docs"},
	}

	virtual := BuildCollections(pages, colCfg)
	if len(virtual) != 1 {
		t.Fatalf("expected 1, got %d", len(virtual))
	}
	env := virtual[0].Envelope
	if env["collection"] != "docs" {
		t.Errorf("expected collection='docs', got %v", env["collection"])
	}
}

func TestBuildCollections_SortFieldInAttributes(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.trivia", URL: "/a/", Type: "trivia", Envelope: core.Envelope{
			"title": "A", "points": 10,
		}},
	}

	colCfg := map[string]config.CollectionConfig{
		"all": {Match: "**", SortBy: "points", Path: "/all"},
	}

	virtual := BuildCollections(pages, colCfg)
	node := virtual[0].Nodes[0]
	if node.Attributes["points"] != 10 {
		t.Errorf("expected points=10 in attributes, got %v", node.Attributes["points"])
	}
}
