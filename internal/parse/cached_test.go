package parse

import (
	"context"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/discover"
)

// treeTestParser expands a file into a fixed 3-page tree (root + two children)
// so the cache's multi-page storage path can be exercised. The root envelope
// carries the raw bytes' length so a content edit changes the cached root.
type treeTestParser struct{}

func (treeTestParser) Type() string { return "tree" }
func (treeTestParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	t, err := treeTestParser{}.ParseTree(raw)
	return t.Envelope, t.Nodes, err
}
func (treeTestParser) ParseTree(raw []byte) (*core.PageTree, error) {
	return &core.PageTree{
		Envelope: core.Envelope{"title": "Root", "size": len(raw)},
		Nodes:    []core.Node{{Type: "tree", Content: "root"}},
		Children: map[string]*core.PageTree{
			"a": {Envelope: core.Envelope{"title": "A"}, Nodes: []core.Node{{Type: "tree", Content: "a"}}},
			"b": {Envelope: core.Envelope{"title": "B"}, Nodes: []core.Node{{Type: "tree", Content: "b"}}},
		},
	}, nil
}

// TestParseAllCachedTreeReuse proves the whole tree is stored under the source
// file's single content-hash entry: an unchanged artifact restores its whole
// tree from cache (Reused, one cache key), a changed artifact reparses exactly
// its tree, and the restored children carry their composite RelPaths.
func TestParseAllCachedTreeReuse(t *testing.T) {
	dir := t.TempDir()
	spec := writeTestFile(t, dir, "api.tree", "v1")
	files := []discover.FileEntry{
		{Path: spec, RelPath: "api.tree", Ext: "tree", MatchedParser: "tree"},
	}
	parsers := map[string]core.Parser{"tree": treeTestParser{}}

	// Cold: parses the file, produces root + 2 children, caches under one key.
	pages, errs, prev, stats := ParseAllCached(context.Background(), files, parsers, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages (root + 2 children), got %d", len(pages))
	}
	if stats.Parsed != 1 || stats.Reused != 0 {
		t.Fatalf("cold build: %+v, want Parsed 1 Reused 0", stats)
	}
	if len(prev) != 1 {
		t.Fatalf("cache map should have ONE entry for the whole tree, got %d", len(prev))
	}
	if got := len(prev["api.tree"].Tree); got != 2 {
		t.Fatalf("cache entry should store 2 tree children, got %d", got)
	}

	// No-op rebuild: the whole tree is restored from the one entry, no reparse.
	pages, _, next, stats := ParseAllCached(context.Background(), files, parsers, prev)
	if stats.Parsed != 0 || stats.Reused != 1 {
		t.Fatalf("no-op build: %+v, want Parsed 0 Reused 1", stats)
	}
	if len(pages) != 3 {
		t.Fatalf("cache hit should restore all 3 pages, got %d", len(pages))
	}
	// Restored children keep their composite RelPaths and tree linkage.
	rels := map[string]bool{}
	for _, p := range pages {
		rels[p.RelPath] = true
		if p.TreeSlugPath != "" && p.TreeRootRel != "api.tree" {
			t.Errorf("restored child %q lost its TreeRootRel: %q", p.RelPath, p.TreeRootRel)
		}
	}
	for _, want := range []string{"api.tree", "api.tree/a", "api.tree/b"} {
		if !rels[want] {
			t.Errorf("restored tree missing page %q", want)
		}
	}

	// Edit the artifact: exactly its tree reparses.
	writeTestFile(t, dir, "api.tree", "v2-longer")
	_, _, _, stats = ParseAllCached(context.Background(), files, parsers, next)
	if stats.Parsed != 1 || stats.Reused != 0 {
		t.Fatalf("after editing the artifact: %+v, want Parsed 1 Reused 0", stats)
	}
}

func TestParseAllCachedReuse(t *testing.T) {
	dir := t.TempDir()
	a := writeTestFile(t, dir, "a.md", "---\ntitle: A\n---\nbody a")
	b := writeTestFile(t, dir, "b.md", "---\ntitle: B\n---\nbody b")
	c := writeTestFile(t, dir, "c.md", "---\ntitle: C\n---\nbody c")

	files := []discover.FileEntry{
		{Path: a, RelPath: "a.md", Ext: "md"},
		{Path: b, RelPath: "b.md", Ext: "md"},
		{Path: c, RelPath: "c.md", Ext: "md"},
	}

	// Cold: everything is parsed, nothing reused.
	pages, errs, prev, stats := ParseAllCached(context.Background(), files, nil, nil)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
	if stats.Parsed != 3 || stats.Reused != 0 {
		t.Fatalf("cold build: %+v, want Parsed 3 Reused 0", stats)
	}

	// Edit only b.md; a and c must be served from cache.
	writeTestFile(t, dir, "b.md", "---\ntitle: B\n---\nbody b edited")

	_, _, next, stats := ParseAllCached(context.Background(), files, nil, prev)
	if stats.Parsed != 1 || stats.Reused != 2 {
		t.Fatalf("after editing one file: %+v, want Parsed 1 Reused 2", stats)
	}
	if len(next) != 3 {
		t.Fatalf("cache map should cover all 3 current files, got %d", len(next))
	}

	// A second build with no change reuses all three.
	_, _, _, stats = ParseAllCached(context.Background(), files, nil, next)
	if stats.Parsed != 0 || stats.Reused != 3 {
		t.Fatalf("no-op build: %+v, want Parsed 0 Reused 3", stats)
	}
}

func TestParseAllCachedDropsDeletedFiles(t *testing.T) {
	dir := t.TempDir()
	a := writeTestFile(t, dir, "a.md", "---\ntitle: A\n---\nx")
	b := writeTestFile(t, dir, "b.md", "---\ntitle: B\n---\ny")

	files := []discover.FileEntry{
		{Path: a, RelPath: "a.md", Ext: "md"},
		{Path: b, RelPath: "b.md", Ext: "md"},
	}
	_, _, prev, _ := ParseAllCached(context.Background(), files, nil, nil)

	// b is gone on the next build: the new cache map must not retain it.
	files = files[:1]
	_, _, next, _ := ParseAllCached(context.Background(), files, nil, prev)
	if _, ok := next["b.md"]; ok {
		t.Error("deleted file should not remain in the cache map")
	}
	if len(next) != 1 {
		t.Errorf("cache map should cover only current files, got %d", len(next))
	}
}
