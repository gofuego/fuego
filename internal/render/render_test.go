package render

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestDefaultRenderer(t *testing.T) {
	nodes := []core.Node{
		{
			Type:    "question",
			Content: "What is Go?",
			Children: []core.Node{
				{Type: "option", Content: "A language", Attributes: map[string]any{"correct": true}},
				{Type: "option", Content: "A framework", Attributes: map[string]any{"correct": false}},
			},
		},
	}

	html := string(DefaultRenderer(nodes))

	if !strings.Contains(html, `data-type="question"`) {
		t.Error("missing data-type for question")
	}
	if !strings.Contains(html, "What is Go?") {
		t.Error("missing question content")
	}
	if !strings.Contains(html, `data-type="option"`) {
		t.Error("missing data-type for option")
	}
	if !strings.Contains(html, "A language") {
		t.Error("missing option content")
	}
	if !strings.Contains(html, `data-attrs=`) {
		t.Error("missing data-attrs")
	}
}

func TestDefaultRendererRawNode(t *testing.T) {
	nodes := []core.Node{
		{Type: "custom-type", Content: "<p>Pre-rendered HTML</p>", Raw: true},
	}
	html := string(DefaultRenderer(nodes))
	if html != "<p>Pre-rendered HTML</p>" {
		t.Errorf("Raw node should pass through without wrapping, got %q", html)
	}
}

func TestDefaultRendererRawFalseWraps(t *testing.T) {
	// Even Type "html" should be wrapped if Raw is false
	nodes := []core.Node{
		{Type: "html", Content: "<p>Hello</p>"},
	}
	html := string(DefaultRenderer(nodes))
	if !strings.Contains(html, `data-type="html"`) {
		t.Error("Non-raw node with Type 'html' should be wrapped in div")
	}
}

func TestDefaultRendererEscapesHTML(t *testing.T) {
	nodes := []core.Node{
		{Type: "text", Content: `<script>alert("xss")</script>`},
	}

	html := string(DefaultRenderer(nodes))

	if strings.Contains(html, "<script>") {
		t.Error("content should be HTML-escaped")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("expected HTML-escaped content")
	}
}

func TestJSONPayload(t *testing.T) {
	env := core.Envelope{"title": "Test", "points": 10}
	nodes := []core.Node{{Type: "raw", Content: "hello"}}

	jsonStr, err := JSONPayload(env, nodes, "/test/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed["url"] != "/test/" {
		t.Errorf("expected url '/test/', got %v", parsed["url"])
	}
}

func TestJSONPayloadEscapesScriptTags(t *testing.T) {
	env := core.Envelope{"title": `</script><script>alert('xss')`}
	nodes := []core.Node{}

	jsonStr, err := JSONPayload(env, nodes, "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// json.Marshal escapes < > & to unicode sequences
	if strings.Contains(jsonStr, "</script>") {
		t.Error("JSON should escape </script> to prevent injection")
	}
}

func setupTheme(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for relPath, content := range files {
		full := filepath.Join(dir, relPath)
		os.MkdirAll(filepath.Dir(full), 0755)
		os.WriteFile(full, []byte(content), 0644)
	}
}

func TestRenderAllSinglePage(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "theme")
	outputDir := filepath.Join(dir, "build")

	setupTheme(t, themeDir, map[string]string{
		"base.html": `<!DOCTYPE html>
<html>
<head><title>{{.Page.Envelope.title}} | {{.Site.Name}}</title></head>
<body data-layout="{{.Page.Layout}}">
<div id="root">{{.Page.Content}}</div>
<script type="application/json" id="fuego-data">{{.JSON}}</script>
</body>
</html>`,
	})

	pages := []*core.Page{
		{
			RelPath:  "hello.md",
			Ext:      "md",
			Envelope: core.Envelope{"title": "Hello World"},
			Nodes:    []core.Node{{Type: "raw", Content: "Welcome to Fuego"}},
			URL:      "/hello/",
			Type:     "md",
		},
	}

	cfg := &config.Config{
		Site: config.SiteConfig{Name: "Test Site"},
		Dirs: config.DirsConfig{Theme: themeDir, Output: outputDir},
	}

	errs := RenderAll(context.Background(), pages, cfg, nil, nil, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	outFile := filepath.Join(outputDir, "hello", "index.html")
	content, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	html := string(content)
	if !strings.Contains(html, "<title>Hello World | Test Site</title>") {
		t.Error("missing title")
	}
	if !strings.Contains(html, "Welcome to Fuego") {
		t.Error("missing content")
	}
	if !strings.Contains(html, `id="fuego-data"`) {
		t.Error("missing JSON data script tag")
	}
}

func TestRenderAllWithLayout(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "theme")
	outputDir := filepath.Join(dir, "build")

	setupTheme(t, themeDir, map[string]string{
		"base.html": `<!DOCTYPE html>
<html><body>
{{block "content" .}}DEFAULT{{end}}
</body></html>`,
		"layouts/quiz.html": `{{define "content"}}<div class="quiz">{{.Page.Content}}</div>{{end}}`,
	})

	pages := []*core.Page{
		{
			RelPath:  "q1.trivia",
			Ext:      "trivia",
			Envelope: core.Envelope{"title": "Q1"},
			Nodes:    []core.Node{{Type: "question", Content: "What is Go?"}},
			URL:      "/q1/",
			Layout:   "quiz",
			Type:     "trivia",
		},
	}

	cfg := &config.Config{
		Dirs: config.DirsConfig{Theme: themeDir, Output: outputDir},
	}

	errs := RenderAll(context.Background(), pages, cfg, nil, nil, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	content, _ := os.ReadFile(filepath.Join(outputDir, "q1", "index.html"))
	html := string(content)

	if !strings.Contains(html, `class="quiz"`) {
		t.Error("layout template should be applied")
	}
	if strings.Contains(html, "DEFAULT") {
		t.Error("default block should be overridden by layout")
	}
}

func TestRenderAllWithCustomRenderer(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "theme")
	outputDir := filepath.Join(dir, "build")

	setupTheme(t, themeDir, map[string]string{
		"base.html":             `<!DOCTYPE html><body>{{.Page.Content}}</body>`,
		"renderers/question.html": `<h2 class="custom">{{.Content}}</h2>`,
	})

	pages := []*core.Page{
		{
			RelPath:  "q1.md",
			Ext:      "md",
			Envelope: core.Envelope{"title": "Q1"},
			Nodes:    []core.Node{{Type: "question", Content: "What is Go?"}},
			URL:      "/q1/",
			Type:     "md",
		},
	}

	cfg := &config.Config{
		Dirs: config.DirsConfig{Theme: themeDir, Output: outputDir},
	}

	errs := RenderAll(context.Background(), pages, cfg, nil, nil, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	content, _ := os.ReadFile(filepath.Join(outputDir, "q1", "index.html"))
	html := string(content)

	if !strings.Contains(html, `class="custom"`) {
		t.Error("custom renderer should be used for 'question' type nodes")
	}
	if strings.Contains(html, `data-type="question"`) {
		t.Error("default renderer should NOT be used when custom renderer exists")
	}
}

func TestCleanOutput(t *testing.T) {
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "build")

	// Create some stale files
	os.MkdirAll(filepath.Join(outputDir, "old"), 0755)
	os.WriteFile(filepath.Join(outputDir, "old", "stale.html"), []byte("stale"), 0644)

	err := CleanOutput(outputDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify old content is gone but dir exists
	entries, _ := os.ReadDir(outputDir)
	if len(entries) != 0 {
		t.Error("output dir should be empty after clean")
	}
}

func TestRenderAllMultiplePages(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "theme")
	outputDir := filepath.Join(dir, "build")

	setupTheme(t, themeDir, map[string]string{
		"base.html": `<!DOCTYPE html><body>{{.Page.Content}}</body>`,
	})

	pages := []*core.Page{
		{RelPath: "a.md", Envelope: core.Envelope{"title": "A"}, Nodes: []core.Node{{Type: "raw", Content: "Page A"}}, URL: "/a/", Type: "md"},
		{RelPath: "b.md", Envelope: core.Envelope{"title": "B"}, Nodes: []core.Node{{Type: "raw", Content: "Page B"}}, URL: "/b/", Type: "md"},
		{RelPath: "c.md", Envelope: core.Envelope{"title": "C"}, Nodes: []core.Node{{Type: "raw", Content: "Page C"}}, URL: "/c/", Type: "md"},
	}

	cfg := &config.Config{
		Dirs: config.DirsConfig{Theme: themeDir, Output: outputDir},
	}

	errs := RenderAll(context.Background(), pages, cfg, nil, nil, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	for _, url := range []string{"/a/", "/b/", "/c/"} {
		outFile := filepath.Join(outputDir, filepath.FromSlash(url), "index.html")
		if _, err := os.Stat(outFile); os.IsNotExist(err) {
			t.Errorf("expected output file at %s", outFile)
		}
	}
}
