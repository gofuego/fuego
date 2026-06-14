package render

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/gofuego/fuego/core"
)

func TestCopyPackStatic(t *testing.T) {
	out := t.TempDir()

	packs := []core.Pack{
		{Name: "a", Theme: fstest.MapFS{
			"base.html":        &fstest.MapFile{Data: []byte("x")},
			"static/style.css": &fstest.MapFile{Data: []byte("a-css")},
			"static/js/app.js": &fstest.MapFile{Data: []byte("a-js")},
		}},
		// Later pack overwrites an earlier pack's file.
		{Name: "b", Theme: fstest.MapFS{
			"static/style.css": &fstest.MapFile{Data: []byte("b-css")},
		}},
		// A pack with no static/ subtree is fine.
		{Name: "c", Theme: fstest.MapFS{"base.html": &fstest.MapFile{Data: []byte("y")}}},
	}

	if err := CopyPackStatic(packs, out); err != nil {
		t.Fatal(err)
	}

	read := func(rel string) string {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		return string(b)
	}

	if got := read("style.css"); got != "b-css" {
		t.Errorf("style.css = %q, want later pack to win (b-css)", got)
	}
	if got := read("js/app.js"); got != "a-js" {
		t.Errorf("nested js/app.js = %q, want a-js", got)
	}
	// base.html is a template, not a static asset — it must not be copied.
	if _, err := os.Stat(filepath.Join(out, "base.html")); !os.IsNotExist(err) {
		t.Error("base.html should not be copied as a static asset")
	}
}

func TestCopyPackStaticNoPacks(t *testing.T) {
	out := t.TempDir()
	if err := CopyPackStatic(nil, out); err != nil {
		t.Fatalf("nil packs should be a no-op, got %v", err)
	}
}
