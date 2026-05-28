package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMinimalConfig(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte("site:\n  name: Test Site\n"), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Site.Name != "Test Site" {
		t.Errorf("expected site name 'Test Site', got %q", cfg.Site.Name)
	}
}

func TestLoadAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte("site:\n  name: X\n"), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Dirs.Content != "content" {
		t.Errorf("expected default content dir 'content', got %q", cfg.Dirs.Content)
	}
	if cfg.Dirs.Theme != "theme" {
		t.Errorf("expected default theme dir 'theme', got %q", cfg.Dirs.Theme)
	}
	if cfg.Dirs.Output != "build" {
		t.Errorf("expected default output dir 'build', got %q", cfg.Dirs.Output)
	}
	if cfg.Dirs.Static != "public" {
		t.Errorf("expected default static dir 'public', got %q", cfg.Dirs.Static)
	}
	if cfg.Dev.Port != 8080 {
		t.Errorf("expected default dev port 8080, got %d", cfg.Dev.Port)
	}
	if cfg.Dev.ProxyPort != 0 {
		t.Errorf("expected default proxy port 0 (disabled), got %d", cfg.Dev.ProxyPort)
	}
}

func TestLoadOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
site:
  name: Custom
  base_url: https://example.com
dirs:
  content: src
  theme: templates
  output: dist
  static: assets
dev:
  port: 9090
  proxy_port: 5000
`), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Dirs.Content != "src" {
		t.Errorf("expected 'src', got %q", cfg.Dirs.Content)
	}
	if cfg.Dirs.Theme != "templates" {
		t.Errorf("expected 'templates', got %q", cfg.Dirs.Theme)
	}
	if cfg.Dirs.Output != "dist" {
		t.Errorf("expected 'dist', got %q", cfg.Dirs.Output)
	}
	if cfg.Dirs.Static != "assets" {
		t.Errorf("expected 'assets', got %q", cfg.Dirs.Static)
	}
	if cfg.Dev.Port != 9090 {
		t.Errorf("expected 9090, got %d", cfg.Dev.Port)
	}
	if cfg.Site.BaseURL != "https://example.com" {
		t.Errorf("expected 'https://example.com', got %q", cfg.Site.BaseURL)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(":::bad yaml\n\t\t{{{"), 0644)

	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadWithParsers(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
site:
  name: Test
parsers:
  card:
    rules:
      - match: "^front:\\s*(.+)$"
        emit:
          type: front
          content: "$1"
      - match: "^back:\\s*(.+)$"
        emit:
          type: back
          content: "$1"
`), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pc, ok := cfg.Parsers["card"]
	if !ok {
		t.Fatal("expected 'card' parser in config")
	}
	if len(pc.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(pc.Rules))
	}
	if pc.Rules[0].Emit.Type != "front" {
		t.Errorf("expected emit type 'front', got %q", pc.Rules[0].Emit.Type)
	}
}

func TestLoadWithCollectionsAndTaxonomies(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
site:
  name: Test
routes:
  card: "/cards/{slug}"
collections:
  all-cards:
    match: "content/*.card"
    sort_by: title
    layout: card-list
    path: "/cards/"
taxonomies:
  tags:
    path: "/tags/{value}"
    layout: tag-term
    index_path: "/tags"
    index_layout: tag-index
`), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Routes["card"] != "/cards/{slug}" {
		t.Errorf("unexpected route: %q", cfg.Routes["card"])
	}

	col, ok := cfg.Collections["all-cards"]
	if !ok {
		t.Fatal("expected 'all-cards' collection")
	}
	if col.SortBy != "title" {
		t.Errorf("expected sort_by 'title', got %q", col.SortBy)
	}

	tax, ok := cfg.Taxonomies["tags"]
	if !ok {
		t.Fatal("expected 'tags' taxonomy")
	}
	if tax.IndexLayout != "tag-index" {
		t.Errorf("expected index_layout 'tag-index', got %q", tax.IndexLayout)
	}
}

func TestNilMapsInitialized(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte("site:\n  name: X\n"), 0644)

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Routes == nil {
		t.Error("Routes map should be initialized")
	}
	if cfg.Collections == nil {
		t.Error("Collections map should be initialized")
	}
	if cfg.Taxonomies == nil {
		t.Error("Taxonomies map should be initialized")
	}
	if cfg.Parsers == nil {
		t.Error("Parsers map should be initialized")
	}
}
