package route

import (
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestResolveFilesystemMirror(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "hello.md", Ext: "md", Envelope: core.Envelope{}},
		{RelPath: "trivia/history/q1.trivia", Ext: "trivia", Envelope: core.Envelope{}},
		{RelPath: "chess/puzzle1.chess", Ext: "chess", Envelope: core.Envelope{}},
	}

	cfg := &config.Config{}
	errs := ResolveAll(pages, cfg)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	expected := map[string]string{
		"hello.md":                     "/hello/",
		"trivia/history/q1.trivia":     "/trivia/history/q1/",
		"chess/puzzle1.chess":          "/chess/puzzle1/",
	}

	for _, p := range pages {
		want, ok := expected[p.RelPath]
		if !ok {
			t.Errorf("unexpected page: %s", p.RelPath)
			continue
		}
		if p.URL != want {
			t.Errorf("%s: got URL %q, want %q", p.RelPath, p.URL, want)
		}
	}
}

func TestResolveSlugOverride(t *testing.T) {
	pages := []*core.Page{
		{
			RelPath:  "trivia/history/nash-eq.trivia",
			Ext:      "trivia",
			Envelope: core.Envelope{"slug": "game-theory-101"},
		},
	}

	cfg := &config.Config{}
	errs := ResolveAll(pages, cfg)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	if pages[0].URL != "/trivia/history/game-theory-101/" {
		t.Errorf("expected slug override URL, got %q", pages[0].URL)
	}
}

func TestDetectCollisions(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "about.md", Ext: "md", Envelope: core.Envelope{}},
		{RelPath: "about.html", Ext: "html", Envelope: core.Envelope{}},
	}

	cfg := &config.Config{}
	errs := ResolveAll(pages, cfg)

	if len(errs) != 1 {
		t.Fatalf("expected 1 collision error, got %d", len(errs))
	}
	if errs[0].Severity != core.GlobalFatal {
		t.Errorf("expected GlobalFatal, got %v", errs[0].Severity)
	}
	if errs[0].Phase != "ROUTE" {
		t.Errorf("expected phase ROUTE, got %q", errs[0].Phase)
	}
}

func TestNoCollisionDifferentPaths(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "a/about.md", Ext: "md", Envelope: core.Envelope{}},
		{RelPath: "b/about.md", Ext: "md", Envelope: core.Envelope{}},
	}

	cfg := &config.Config{}
	errs := ResolveAll(pages, cfg)
	if len(errs) != 0 {
		t.Errorf("expected no collisions, got %v", errs)
	}
}

func TestResolveIndexFile(t *testing.T) {
	// An index file is the root of its directory: content/index.md → "/",
	// content/blog/index.md → "/blog/".
	pages := []*core.Page{
		{RelPath: "index.md", Ext: "md", Envelope: core.Envelope{}},
		{RelPath: "blog/index.md", Ext: "md", Envelope: core.Envelope{}},
		{RelPath: "blog/post.md", Ext: "md", Envelope: core.Envelope{}},
	}

	cfg := &config.Config{}
	ResolveAll(pages, cfg)

	want := []string{"/", "/blog/", "/blog/post/"}
	for i, w := range want {
		if pages[i].URL != w {
			t.Errorf("page %d (%s): expected %q, got %q", i, pages[i].RelPath, w, pages[i].URL)
		}
	}
}

// --- Phase 4: Route pattern tests ---

func TestRoutePattern_DirAndSlug(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "history/nash-eq.trivia", Ext: "trivia", Type: "trivia", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{dir}/{slug}"},
	}
	errs := ResolveAll(pages, cfg)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if pages[0].URL != "/quiz/history/nash-eq/" {
		t.Errorf("got %q, want /quiz/history/nash-eq/", pages[0].URL)
	}
}

func TestRoutePattern_NestedDir(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "science/physics/gravity.trivia", Ext: "trivia", Type: "trivia", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{dir}/{slug}"},
	}
	ResolveAll(pages, cfg)
	if pages[0].URL != "/quiz/science/physics/gravity/" {
		t.Errorf("got %q, want /quiz/science/physics/gravity/", pages[0].URL)
	}
}

func TestRoutePattern_RootLevel_EmptyDir(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "basics.trivia", Ext: "trivia", Type: "trivia", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{dir}/{slug}"},
	}
	ResolveAll(pages, cfg)
	// {dir} is empty at root → double slash collapsed
	if pages[0].URL != "/quiz/basics/" {
		t.Errorf("got %q, want /quiz/basics/", pages[0].URL)
	}
}

func TestRoutePattern_FilenameExpansion(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "history/nash-eq.trivia", Ext: "trivia", Type: "trivia",
			Envelope: core.Envelope{"slug": "game-theory"}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{filename}"},
	}
	ResolveAll(pages, cfg)
	// {filename} always uses the raw filename, not the slug
	if pages[0].URL != "/quiz/nash-eq/" {
		t.Errorf("got %q, want /quiz/nash-eq/", pages[0].URL)
	}
}

func TestRoutePattern_SlugOverrideWithPattern(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "history/nash-eq.trivia", Ext: "trivia", Type: "trivia",
			Envelope: core.Envelope{"slug": "game-theory"}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{dir}/{slug}"},
	}
	ResolveAll(pages, cfg)
	// {slug} uses frontmatter slug when present
	if pages[0].URL != "/quiz/history/game-theory/" {
		t.Errorf("got %q, want /quiz/history/game-theory/", pages[0].URL)
	}
}

func TestRoutePattern_MissingPattern_FallbackToMirror(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "docs/readme.md", Ext: "md", Type: "md", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{slug}"},
	}
	ResolveAll(pages, cfg)
	// md has no route pattern → falls back to filesystem mirror
	if pages[0].URL != "/docs/readme/" {
		t.Errorf("got %q, want /docs/readme/", pages[0].URL)
	}
}

func TestRoutePattern_CollisionBetweenPatternResolved(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "a/q1.trivia", Ext: "trivia", Type: "trivia", Envelope: core.Envelope{}},
		{RelPath: "b/q1.trivia", Ext: "trivia", Type: "trivia", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"trivia": "/quiz/{slug}"},
	}
	errs := ResolveAll(pages, cfg)
	if len(errs) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(errs))
	}
	if errs[0].Severity != core.GlobalFatal {
		t.Errorf("expected GlobalFatal, got %v", errs[0].Severity)
	}
}

func TestRoutePattern_SlugOnlyPattern(t *testing.T) {
	pages := []*core.Page{
		{RelPath: "deep/nested/file.card", Ext: "card", Type: "card", Envelope: core.Envelope{}},
	}
	cfg := &config.Config{
		Routes: map[string]string{"card": "/{slug}"},
	}
	ResolveAll(pages, cfg)
	if pages[0].URL != "/file/" {
		t.Errorf("got %q, want /file/", pages[0].URL)
	}
}
