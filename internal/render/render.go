package render

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"golang.org/x/sync/errgroup"
)

// TemplateData is the data passed to each page template.
type TemplateData struct {
	Page      PageTemplateData
	Site      SiteTemplateData
	Paginator *core.Paginator // set on paginated listing pages, nil otherwise
	JSON      template.JS     // raw JSON blob, safe for embedding in <script>
}

// PageTemplateData contains the current page's data for templates.
type PageTemplateData struct {
	Envelope core.Envelope
	Content  template.HTML
	URL      string
	Layout   string
	Type     string
}

// SiteTemplateData contains global site data for templates.
type SiteTemplateData struct {
	Name    string
	BaseURL string
	Pages   []PageRef
}

// PageRef is the slim cross-page reference exposed as .Site.Pages. It
// deliberately omits Nodes and rendered content: refs exist for navs,
// listings, and envelope-driven queries, not for rendering other pages'
// bodies (which would make builds O(n²)).
type PageRef struct {
	URL      string
	Type     string
	Layout   string
	Envelope core.Envelope
}

// BuildPageRefs projects pages into URL-sorted refs. Pages marked Skip are
// excluded. The result is built once per build and shared read-only across
// render workers.
func BuildPageRefs(pages []*core.Page) []PageRef {
	refs := make([]PageRef, 0, len(pages))
	for _, p := range pages {
		if p.Skip {
			continue
		}
		refs = append(refs, PageRef{
			URL:      p.URL,
			Type:     p.Type,
			Layout:   p.Layout,
			Envelope: p.Envelope,
		})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].URL < refs[j].URL })
	return refs
}

// RenderAll renders pages to HTML files in the output directory, in parallel.
//
// On a narrowed incremental rebuild (narrow=true), only the affected set is
// re-rendered: pages whose content changed (changed[RelPath]), all virtual
// pages (they aggregate content), and pages whose template reads .Site.Pages.
// Site-blind, unchanged pages keep their existing output. With narrow=false,
// every page is rendered. The .Site.Pages refs are always built from the full
// page set.
func RenderAll(ctx context.Context, pages []*core.Page, cfg *config.Config, packs []core.Pack, changed map[string]bool, narrow bool) []core.EngineError {
	themeDir := cfg.Dirs.Theme
	outputDir := cfg.Dirs.Output

	tc, err := LoadTemplates(themeDir, packs)
	if err != nil {
		return []core.EngineError{{
			Phase:    "RENDER",
			Severity: core.GlobalFatal,
			Err:      fmt.Errorf("loading templates: %w", err),
		}}
	}

	site := SiteTemplateData{
		Name:    cfg.Site.Name,
		BaseURL: cfg.Site.BaseURL,
		Pages:   BuildPageRefs(pages),
	}

	errs := make([]core.EngineError, len(pages))
	hasErr := make([]bool, len(pages))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for i, page := range pages {
		idx := i
		p := page

		if narrow && !affected(p, tc, changed) {
			continue
		}

		g.Go(func() error {
			if engErr := renderPage(p, tc, site, outputDir); engErr != nil {
				errs[idx] = *engErr
				hasErr[idx] = true
			}
			return nil
		})
	}

	g.Wait()

	var validErrs []core.EngineError
	for i := range pages {
		if hasErr[i] {
			validErrs = append(validErrs, errs[i])
		}
	}
	return validErrs
}

// affected reports whether a page must be re-rendered on a narrowed rebuild.
func affected(p *core.Page, tc *TemplateCache, changed map[string]bool) bool {
	if p.SourcePath == "" {
		return true // virtual page — aggregates content, always re-render
	}
	if changed[p.RelPath] {
		return true // content changed
	}
	return tc.UsesSitePages(tc.GetLayout(p.Layout)) // depends on the site page list
}

func renderPage(page *core.Page, tc *TemplateCache, site SiteTemplateData, outputDir string) *core.EngineError {
	// Select template first: the JSON payload is only serialized when the
	// resolved template actually references .JSON.
	tmpl := tc.GetLayout(page.Layout)

	// Generate pre-rendered content HTML
	content := tc.renderWithOverrides(page.Nodes)

	// Generate JSON payload for client-side hydration
	var jsonStr string
	if tc.UsesJSON(tmpl) {
		var err error
		jsonStr, err = JSONPayload(page.Envelope, page.Nodes, page.URL)
		if err != nil {
			return &core.EngineError{
				Phase:    "RENDER",
				File:     page.RelPath,
				Severity: core.LocalFatal,
				Err:      err,
			}
		}
	}

	data := TemplateData{
		Page: PageTemplateData{
			Envelope: page.Envelope,
			Content:  content,
			URL:      page.URL,
			Layout:   page.Layout,
			Type:     page.Type,
		},
		Site:      site,
		Paginator: page.Paginator,
		JSON:      template.JS(jsonStr),
	}

	// Determine output path
	outPath := filepath.Join(outputDir, filepath.FromSlash(page.URL), "index.html")

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return &core.EngineError{
			Phase:    "RENDER",
			File:     page.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("creating output directory: %w", err),
		}
	}

	// Render to file
	f, err := os.Create(outPath)
	if err != nil {
		return &core.EngineError{
			Phase:    "RENDER",
			File:     page.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("creating output file: %w", err),
		}
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		layoutName := page.Layout
		if layoutName == "" {
			layoutName = "base"
		}
		return &core.EngineError{
			Phase:    "RENDER",
			File:     page.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("executing layout %q: %w", layoutName, err),
		}
	}

	return nil
}

// CleanOutput removes and recreates the output directory for a fresh build.
func CleanOutput(outputDir string) error {
	// Remove if exists (ignore error if doesn't exist)
	os.RemoveAll(outputDir)
	return os.MkdirAll(outputDir, 0755)
}

// BuildOutputPath returns the output directory, resolving relative paths
// against the current working directory.
func BuildOutputPath(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	// Normalize trailing path separators
	return strings.TrimRight(abs, string(filepath.Separator)), nil
}
