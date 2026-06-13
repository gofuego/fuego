package buildcache

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/FabioSol/fuego/core"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.bin")

	c := New(Header{BinaryID: "bin", ConfigHash: "cfg", ThemeHash: "thm"})
	c.Pages["a.md"] = ParsedPage{
		ContentHash: "h1",
		Envelope:    core.Envelope{"title": "A", "weight": 3, "tags": []any{"go", "web"}},
		Nodes:       []core.Node{{Type: "html", Content: "<p>hi</p>", Raw: true}},
		Type:        "md",
	}
	c.Outputs = []string{"a/index.html"}

	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	got, ok := Load(path)
	if !ok {
		t.Fatal("expected cache to load")
	}
	if !reflect.DeepEqual(got.Pages["a.md"], c.Pages["a.md"]) {
		t.Errorf("round-trip mismatch:\n got %#v\nwant %#v", got.Pages["a.md"], c.Pages["a.md"])
	}
	// Envelope int must survive as int, not float64 (gob preserves types).
	if _, ok := got.Pages["a.md"].Envelope["weight"].(int); !ok {
		t.Errorf("weight type not preserved: %T", got.Pages["a.md"].Envelope["weight"])
	}
}

func TestValidHeader(t *testing.T) {
	h := Header{BinaryID: "bin", ConfigHash: "cfg", ThemeHash: "thm"}
	c := New(h)

	if !c.Valid(h) {
		t.Error("cache should be valid for the same header")
	}
	for _, bad := range []Header{
		{BinaryID: "other", ConfigHash: "cfg", ThemeHash: "thm"},
		{BinaryID: "bin", ConfigHash: "other", ThemeHash: "thm"},
		{BinaryID: "bin", ConfigHash: "cfg", ThemeHash: "other"},
	} {
		if c.Valid(bad) {
			t.Errorf("cache should be invalid for changed header %+v", bad)
		}
	}
}

func TestVersionMismatchInvalidates(t *testing.T) {
	h := Header{BinaryID: "bin", ConfigHash: "cfg", ThemeHash: "thm"}
	c := New(h)
	c.Header.Version = cacheVersion + 1 // simulate an older/newer on-disk format
	if c.Valid(h) {
		t.Error("a version mismatch must invalidate the cache")
	}
}

func TestLoadMissingIsCacheMiss(t *testing.T) {
	c, ok := Load(filepath.Join(t.TempDir(), "does-not-exist.bin"))
	if ok {
		t.Error("missing cache should report ok=false")
	}
	if c == nil || c.Pages == nil {
		t.Error("missing cache should still return a usable empty cache")
	}
}

func TestOrphanedOutputs(t *testing.T) {
	old := []string{"a/index.html", "b/index.html", "c/index.html"}
	cur := []string{"a/index.html", "c/index.html", "d/index.html"}
	got := OrphanedOutputs(old, cur)
	if !reflect.DeepEqual(got, []string{"b/index.html"}) {
		t.Errorf("orphans = %v, want [b/index.html]", got)
	}
}

func TestOutputRelPath(t *testing.T) {
	for url, want := range map[string]string{
		"/about/":          "about/index.html",
		"/":                "index.html",
		"/docs/cli/":       "docs/cli/index.html",
		"/blog/page/2/":    "blog/page/2/index.html",
	} {
		if got := OutputRelPath(url); got != want {
			t.Errorf("OutputRelPath(%q) = %q, want %q", url, got, want)
		}
	}
}
