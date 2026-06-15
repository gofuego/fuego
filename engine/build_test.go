package engine

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/parsers/markdown"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// demoPack supplies a theme (incl. a static asset) and a route default, the
// way a real format pack would.
func demoPack() core.Pack {
	return core.Pack{
		Name:  "demo",
		Theme: fstest.MapFS{
			"base.html": &fstest.MapFile{Data: []byte(
				`<!DOCTYPE html><html><head><title>{{.Page.Envelope.title}} | {{.Site.Name}}</title>` +
					`<link rel="stylesheet" href="{{.Site.BaseURL}}/style.css"></head>` +
					`<body>{{block "content" .}}<main>{{.Page.Content}}</main>{{end}}</body></html>`)},
			"static/style.css": &fstest.MapFile{Data: []byte("/* demo */\n")},
		},
		ConfigDefaults: []byte("taxonomies:\n  tags:\n    path: /tags/{term}\n    layout: tag\n    index_path: /tags\n    index_layout: tag\n"),
	}
}

func TestEngineBuildProgrammatic(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "adr")
	outputDir := filepath.Join(dir, "out")
	writeFile(t, filepath.Join(contentDir, "index.md"), "---\ntitle: Home\ntags: [go]\n---\n# Hello\n")

	eng := New()
	eng.Register(markdown.Parser())
	eng.Use(demoPack())

	err := eng.Build(context.Background(), BuildOptions{
		ContentDir: contentDir,
		OutputDir:  outputDir,
		SiteName:   "Programmatic",
		BaseURL:    "/base",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// Page rendered through the pack theme, with the option-set site name.
	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(home), "Home | Programmatic") {
		t.Errorf("site name override not applied:\n%s", home)
	}
	if !strings.Contains(string(home), `href="/base/style.css"`) {
		t.Errorf("base_url override not applied:\n%s", home)
	}
	// Pack static asset copied to output.
	if _, err := os.Stat(filepath.Join(outputDir, "style.css")); err != nil {
		t.Errorf("pack static asset missing: %v", err)
	}
	// Pack-contributed taxonomy produced a term page.
	if _, err := os.Stat(filepath.Join(outputDir, "tags", "go", "index.html")); err != nil {
		t.Errorf("pack taxonomy term page missing: %v", err)
	}
}

func TestEngineValidateProgrammatic(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "c")
	writeFile(t, filepath.Join(contentDir, "a.md"), "---\ntitle: A\n---\nx\n")
	writeFile(t, filepath.Join(contentDir, "b.md"), "---\ntitle: B\n---\ny\n")

	eng := New()
	eng.Register(markdown.Parser())
	eng.Use(demoPack())

	n, err := eng.Validate(context.Background(), BuildOptions{ContentDir: contentDir, OutputDir: filepath.Join(dir, "o")})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if n != 2 {
		t.Errorf("validated %d pages, want 2", n)
	}
}

func TestEngineBuildConfigFileThenOverrides(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "c")
	writeFile(t, filepath.Join(contentDir, "index.md"), "---\ntitle: Home\n---\nx\n")
	cfgPath := filepath.Join(dir, "config.yaml")
	writeFile(t, cfgPath, "site:\n  name: FromFile\n  base_url: /file\n")

	eng := New()
	eng.Register(markdown.Parser())
	eng.Use(demoPack())

	// Options override the file's site name; base_url falls through from the file.
	err := eng.Build(context.Background(), BuildOptions{
		ConfigPath: cfgPath,
		ContentDir: contentDir,
		OutputDir:  filepath.Join(dir, "out"),
		SiteName:   "FromOptions",
	})
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.ReadFile(filepath.Join(dir, "out", "index.html"))
	if !strings.Contains(string(home), "Home | FromOptions") {
		t.Errorf("option should override file site name:\n%s", home)
	}
	if !strings.Contains(string(home), `href="/file/style.css"`) {
		t.Errorf("file base_url should survive when not overridden:\n%s", home)
	}
}
