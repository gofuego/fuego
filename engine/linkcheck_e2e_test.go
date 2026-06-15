package engine_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/engine"
	"github.com/gofuego/fuego/parsers/markdown"
)

// writeSite lays out a minimal Markdown site (content + theme) and returns its
// root. base.html sets <base href> from the site base URL and includes an
// optional extra link in the shell (to exercise template-generated links,
// which the engine does not base-URL-rewrite).
func writeSite(t *testing.T, shellLink string, pages map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range pages {
		p := filepath.Join(root, "content", filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	base := `<!DOCTYPE html><html><head><base href="{{.Site.BaseURL}}/"></head><body>` +
		shellLink +
		`{{block "content" .}}<main>{{.Page.Content}}</main>{{end}}</body></html>`
	themeDir := filepath.Join(root, "theme")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "base.html"), []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func buildOpts(root string, strict bool) engine.BuildOptions {
	return engine.BuildOptions{
		ContentDir:  filepath.Join(root, "content"),
		ThemeDir:    filepath.Join(root, "theme"),
		OutputDir:   filepath.Join(root, "out"),
		BaseURL:     "/owner/repo",
		StrictLinks: strict,
	}
}

func TestBuild_StrictLinks_CatchesBrokenLinks(t *testing.T) {
	// Internal links resolve against <base href> (the site root), so the correct
	// form is base-relative ("about/"). A root-absolute link ("/escape/") escapes
	// the deployment base — the exact bug this guards — and "missing/" is dangling.
	root := writeSite(t, `<a href="/escape/">shell</a>`, map[string]string{
		"index.md": "---\ntitle: Home\n---\n[ok](about/) and [dangling](missing/)\n",
		"about.md": "---\ntitle: About\n---\nhi\n",
	})

	eng := engine.New()
	eng.Register(markdown.Parser())

	err := eng.Build(context.Background(), buildOpts(root, true))
	if err == nil {
		t.Fatal("expected --strict-links build to fail on broken links, got nil")
	}
	if !strings.Contains(err.Error(), "LINKS") {
		t.Errorf("error should mention the LINKS phase: %v", err)
	}
}

func TestBuild_StrictLinks_PassesCleanSite(t *testing.T) {
	// Base-relative links resolve against <base href> and work under any base URL.
	// index.md routes to the site root "/", so "home" is "." (the base href).
	root := writeSite(t, `<a href="about/">nav</a>`, map[string]string{
		"index.md": "---\ntitle: Home\n---\n[about](about/)\n",
		"about.md": "---\ntitle: About\n---\n[home](.)\n",
	})

	eng := engine.New()
	eng.Register(markdown.Parser())

	if err := eng.Build(context.Background(), buildOpts(root, true)); err != nil {
		t.Fatalf("clean site failed strict link check: %v", err)
	}
}
