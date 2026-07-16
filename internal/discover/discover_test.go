package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
	"github.com/gofuego/fuego/internal/dispatch"
)

// stubParser claims a bare extension (its Type) and optionally filename
// patterns, standing in for a registered parser during discovery tests.
type stubParser struct {
	typ      string
	patterns []string
}

func (p stubParser) Type() string                                     { return p.typ }
func (p stubParser) Parse([]byte) (core.Envelope, []core.Node, error) { return nil, nil, nil }
func (p stubParser) Filenames() []string                              { return p.patterns }

// resolverFor builds a dispatch resolver from stub parsers, so discovery tests
// exercise the same claim rule the pipeline wires in production.
func resolverFor(parsers ...core.Parser) *dispatch.Resolver {
	return dispatch.NewResolver(parsers)
}

func setupContentDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	for relPath, content := range files {
		full := filepath.Join(contentDir, relPath)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte(content), 0644)
	}
	return contentDir
}

func TestWalkSingleFile(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"hello.md": "# Hello",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "md"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].RelPath != "hello.md" {
		t.Errorf("expected relPath 'hello.md', got %q", entries[0].RelPath)
	}
	if entries[0].Ext != "md" {
		t.Errorf("expected ext 'md', got %q", entries[0].Ext)
	}
	if entries[0].IsAsset {
		t.Error("md file should be content when parser is registered")
	}
}

func TestWalkNoParserRegistered(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"hello.md": "# Hello",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if !entries[0].IsAsset {
		t.Error("md file should be asset when no parser is registered")
	}
}

func TestWalkNestedDirectories(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"trivia/history/q1.md": "# Q1",
		"trivia/science/q2.md": "# Q2",
		"chess/p1.md":          "# P1",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "md"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestWalkAssetsVsContent(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"article.md":    "# Article",
		"img/photo.png": "fake png",
		"data.json":     "{}",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "md"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assetCount := 0
	contentCount := 0
	for _, e := range entries {
		if e.IsAsset {
			assetCount++
		} else {
			contentCount++
		}
	}
	if contentCount != 1 {
		t.Errorf("expected 1 content file, got %d", contentCount)
	}
	if assetCount != 2 {
		t.Errorf("expected 2 asset files, got %d", assetCount)
	}
}

func TestWalkWithRegisteredTypes(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"q1.trivia": "question data",
		"p1.chess":  "chess data",
		"photo.png": "fake png",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "trivia"}, stubParser{typ: "chess"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, e := range entries {
		switch e.Ext {
		case "trivia", "chess":
			if e.IsAsset {
				t.Errorf("%s should be content with registered type", e.RelPath)
			}
		case "png":
			if !e.IsAsset {
				t.Errorf("png should be asset")
			}
		}
	}
}

func TestWalkEmptyDir(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	os.MkdirAll(contentDir, 0755)

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty dir, got %d", len(entries))
	}
}

func TestWalkFilenamePattern(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"Dockerfile":     "FROM golang:1.22",
		"app/Dockerfile": "FROM node:18",
		"readme.md":      "# Readme",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "dockerfile", patterns: []string{"Dockerfile"}}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contentCount := 0
	for _, e := range entries {
		if !e.IsAsset {
			contentCount++
			if e.MatchedParser != "dockerfile" {
				t.Errorf("expected MatchedParser 'dockerfile', got %q", e.MatchedParser)
			}
		}
	}
	if contentCount != 2 {
		t.Errorf("expected 2 Dockerfile content files, got %d", contentCount)
	}
}

func TestWalkFilenamePatternWildcard(t *testing.T) {
	contentDir := setupContentDir(t, map[string]string{
		"Dockerfile":       "FROM golang",
		"Dockerfile.prod":  "FROM golang",
		"Makefile":         "all:",
	})

	cfg := &config.Config{Dirs: config.DirsConfig{Content: contentDir}}
	entries, err := Walk(cfg, resolverFor(stubParser{typ: "dockerfile", patterns: []string{"Dockerfile*"}}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	matched := 0
	for _, e := range entries {
		if !e.IsAsset {
			matched++
		}
	}
	if matched != 2 {
		t.Errorf("expected 2 matched files (Dockerfile, Dockerfile.prod), got %d", matched)
	}
}

func TestWalkMissingDir(t *testing.T) {
	cfg := &config.Config{Dirs: config.DirsConfig{Content: "/nonexistent/content"}}
	_, err := Walk(cfg, resolverFor())
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}
