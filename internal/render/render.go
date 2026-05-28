package render

import (
	"context"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"golang.org/x/sync/errgroup"
)

// TemplateData is the data passed to each page template.
type TemplateData struct {
	Page PageTemplateData
	Site SiteTemplateData
	JSON template.JS // raw JSON blob, safe for embedding in <script>
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
}

// RenderAll renders all pages to HTML files in the output directory.
// Uses errgroup for parallel rendering.
func RenderAll(ctx context.Context, pages []*core.Page, cfg *config.Config) []core.EngineError {
	themeDir := cfg.Dirs.Theme
	outputDir := cfg.Dirs.Output

	tc, err := LoadTemplates(themeDir)
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
	}

	errs := make([]core.EngineError, len(pages))
	hasErr := make([]bool, len(pages))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for i, page := range pages {
		idx := i
		p := page

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

func renderPage(page *core.Page, tc *TemplateCache, site SiteTemplateData, outputDir string) *core.EngineError {
	// Generate pre-rendered content HTML
	content := tc.renderWithOverrides(page.Nodes)

	// Generate JSON payload for client-side hydration
	jsonStr, err := JSONPayload(page.Envelope, page.Nodes, page.URL)
	if err != nil {
		return &core.EngineError{
			Phase:    "RENDER",
			File:     page.RelPath,
			Severity: core.LocalFatal,
			Err:      err,
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
		Site: site,
		JSON: template.JS(jsonStr),
	}

	// Select template
	tmpl := tc.GetLayout(page.Layout)

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
		return &core.EngineError{
			Phase:    "RENDER",
			File:     page.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("executing template: %w", err),
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
