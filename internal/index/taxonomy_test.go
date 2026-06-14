package index

import (
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestBuildTaxonomies_BasicTermPages(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "Post A", "tags": []any{"go", "web"},
		}},
		{RelPath: "b.md", URL: "/b/", Type: "md", Envelope: core.Envelope{
			"title": "Post B", "tags": []any{"go"},
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}", Layout: "tag", IndexPath: "/tags", IndexLayout: "tag-index"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)

	// Should produce: /tags/go/ (2 members), /tags/web/ (1 member), /tags/ (index)
	if len(virtual) != 3 {
		t.Fatalf("expected 3 virtual pages, got %d", len(virtual))
	}

	// Term pages are sorted alphabetically: go, web
	goPage := virtual[0]
	if goPage.URL != "/tags/go/" {
		t.Errorf("expected /tags/go/, got %q", goPage.URL)
	}
	if goPage.Layout != "tag" {
		t.Errorf("expected layout 'tag', got %q", goPage.Layout)
	}
	if goPage.Type != "taxonomy-term" {
		t.Errorf("expected type 'taxonomy-term', got %q", goPage.Type)
	}
	if len(goPage.Nodes) != 2 {
		t.Errorf("expected 2 page-ref nodes for 'go', got %d", len(goPage.Nodes))
	}

	webPage := virtual[1]
	if webPage.URL != "/tags/web/" {
		t.Errorf("expected /tags/web/, got %q", webPage.URL)
	}
	if len(webPage.Nodes) != 1 {
		t.Errorf("expected 1 page-ref node for 'web', got %d", len(webPage.Nodes))
	}

	// Index page
	indexPage := virtual[2]
	if indexPage.URL != "/tags/" {
		t.Errorf("expected /tags/, got %q", indexPage.URL)
	}
	if indexPage.Layout != "tag-index" {
		t.Errorf("expected layout 'tag-index', got %q", indexPage.Layout)
	}
	if indexPage.Type != "taxonomy-index" {
		t.Errorf("expected type 'taxonomy-index', got %q", indexPage.Type)
	}
	if len(indexPage.Nodes) != 2 {
		t.Errorf("expected 2 term-ref nodes, got %d", len(indexPage.Nodes))
	}
}

func TestBuildTaxonomies_TermNormalization(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "tags": []any{"Go"},
		}},
		{RelPath: "b.md", URL: "/b/", Type: "md", Envelope: core.Envelope{
			"title": "B", "tags": []any{"go"},
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}", IndexPath: "/tags"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	// "Go" and "go" should be normalized to the same term
	// 1 term page + 1 index page = 2
	if len(virtual) != 2 {
		t.Fatalf("expected 2 virtual pages (normalized), got %d", len(virtual))
	}
	if len(virtual[0].Nodes) != 2 {
		t.Errorf("expected 2 members for normalized 'go', got %d", len(virtual[0].Nodes))
	}
}

func TestBuildTaxonomies_SingleStringTerm(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "category": "tutorials",
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"category": {Path: "/category/{term}", IndexPath: "/categories"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	if len(virtual) != 2 {
		t.Fatalf("expected 2 virtual pages, got %d", len(virtual))
	}
	if virtual[0].URL != "/category/tutorials/" {
		t.Errorf("expected /category/tutorials/, got %q", virtual[0].URL)
	}
}

func TestBuildTaxonomies_NoMatchingField(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A",
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}", IndexPath: "/tags"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	if len(virtual) != 0 {
		t.Errorf("expected 0 virtual pages when no pages have the field, got %d", len(virtual))
	}
}

func TestBuildTaxonomies_NoIndexPath(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "tags": []any{"go"},
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	// Only term page, no index page
	if len(virtual) != 1 {
		t.Fatalf("expected 1 virtual page (no index), got %d", len(virtual))
	}
}

func TestBuildTaxonomies_TermPageEnvelope(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "tags": []any{"go"},
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	env := virtual[0].Envelope
	if env["taxonomy"] != "tags" {
		t.Errorf("expected taxonomy='tags', got %v", env["taxonomy"])
	}
	if env["term"] != "go" {
		t.Errorf("expected term='go', got %v", env["term"])
	}
}

func TestBuildTaxonomies_IndexPageTermRefs(t *testing.T) {
	t.Parallel()

	pages := []*core.Page{
		{RelPath: "a.md", URL: "/a/", Type: "md", Envelope: core.Envelope{
			"title": "A", "tags": []any{"go", "web"},
		}},
		{RelPath: "b.md", URL: "/b/", Type: "md", Envelope: core.Envelope{
			"title": "B", "tags": []any{"go"},
		}},
	}

	taxCfg := map[string]config.TaxonomyConfig{
		"tags": {Path: "/tags/{term}", IndexPath: "/tags"},
	}

	virtual := BuildTaxonomies(pages, taxCfg)
	indexPage := virtual[len(virtual)-1]

	// term-ref nodes should have term, count, url
	goRef := indexPage.Nodes[0]
	if goRef.Attributes["term"] != "go" {
		t.Errorf("expected term='go', got %v", goRef.Attributes["term"])
	}
	if goRef.Attributes["count"] != 2 {
		t.Errorf("expected count=2, got %v", goRef.Attributes["count"])
	}
	if goRef.Attributes["url"] != "/tags/go/" {
		t.Errorf("expected url='/tags/go/', got %v", goRef.Attributes["url"])
	}

	webRef := indexPage.Nodes[1]
	if webRef.Attributes["count"] != 1 {
		t.Errorf("expected count=1, got %v", webRef.Attributes["count"])
	}
}
