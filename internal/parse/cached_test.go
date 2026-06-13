package parse

import (
	"context"
	"testing"

	"github.com/FabioSol/fuego/internal/discover"
)

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
