package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func outputCfg(t *testing.T, themeFiles map[string]string) (*config.Config, string) {
	t.Helper()
	themeDir := writeTheme(t, themeFiles)
	outDir := t.TempDir()
	return &config.Config{
		Site: config.SiteConfig{Name: "T", BaseURL: "https://x.test"},
		Dirs: config.DirsConfig{Theme: themeDir, Output: outDir},
	}, outDir
}

func TestRenderOutputsWritesFiles(t *testing.T) {
	cfg, outDir := outputCfg(t, map[string]string{
		"base.html":                  `<body>{{.Page.Content}}</body>`,
		"outputs/robots.txt":         "User-agent: *\nSitemap: {{.Site.BaseURL}}/sitemap.xml\n",
		"outputs/feeds/all.xml":      `<rss>{{range .Site.Pages}}<i>{{.URL}}</i>{{end}}</rss>`,
	})

	pages := []*core.Page{
		{URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
		{URL: "/b/", Type: "md", Envelope: core.Envelope{"title": "B"}},
	}

	if errs := RenderOutputs(pages, cfg, nil); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	robots, err := os.ReadFile(filepath.Join(outDir, "robots.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(robots), "Sitemap: https://x.test/sitemap.xml") {
		t.Errorf("robots.txt: %q", robots)
	}

	// Nested output path preserved.
	feed, err := os.ReadFile(filepath.Join(outDir, "feeds", "all.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(feed) != `<rss><i>/a/</i><i>/b/</i></rss>` {
		t.Errorf("feed: %q", feed)
	}
}

func TestRenderOutputsNoEscaping(t *testing.T) {
	// text/template must not HTML-escape & or < in URLs/content.
	cfg, outDir := outputCfg(t, map[string]string{
		"base.html":         `<body>{{.Page.Content}}</body>`,
		"outputs/feed.xml":  `<link>{{(first .Site.Pages).URL}}</link>`,
	})
	pages := []*core.Page{{URL: "/p/?a=1&b=2/", Type: "md"}}

	if errs := RenderOutputs(pages, cfg, nil); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	out, _ := os.ReadFile(filepath.Join(outDir, "feed.xml"))
	if string(out) != `<link>/p/?a=1&b=2/</link>` {
		t.Errorf("text output should not be HTML-escaped, got %q", out)
	}
}

func TestRenderOutputsCollision(t *testing.T) {
	cfg, _ := outputCfg(t, map[string]string{
		"base.html":               `<body>{{.Page.Content}}</body>`,
		"outputs/about/index.html": "collides",
	})
	pages := []*core.Page{{URL: "/about/", RelPath: "about.md", Type: "md"}}

	errs := RenderOutputs(pages, cfg, nil)
	if len(errs) != 1 || errs[0].Severity != core.GlobalFatal {
		t.Fatalf("expected one GlobalFatal collision, got %v", errs)
	}
	if !strings.Contains(errs[0].Err.Error(), "collides with page") {
		t.Errorf("collision message: %v", errs[0].Err)
	}
}

func TestRenderOutputsNone(t *testing.T) {
	cfg, _ := outputCfg(t, map[string]string{
		"base.html": `<body>{{.Page.Content}}</body>`,
	})
	if errs := RenderOutputs(nil, cfg, nil); errs != nil {
		t.Errorf("no outputs dir should yield no errors, got %v", errs)
	}
}
