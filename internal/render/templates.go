package render

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// TemplateCache holds all pre-parsed templates for rendering.
type TemplateCache struct {
	base      *template.Template
	layouts   map[string]*template.Template
	renderers map[string]*template.Template
	partials  map[string]*template.Template
	funcMap   template.FuncMap
	usesJSON  map[*template.Template]bool
}

// LoadTemplates scans the theme directory and pre-parses all templates.
// It builds the template cache used by the render phase.
func LoadTemplates(themeDir string) (*TemplateCache, error) {
	tc := &TemplateCache{
		layouts:   make(map[string]*template.Template),
		renderers: make(map[string]*template.Template),
		partials:  make(map[string]*template.Template),
	}

	tc.funcMap = tc.buildFuncMap()

	// Load base.html
	basePath := filepath.Join(themeDir, "base.html")
	baseContent, err := os.ReadFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("reading base template: %w", err)
	}

	tc.base, err = template.New("base.html").Funcs(tc.funcMap).Parse(string(baseContent))
	if err != nil {
		return nil, fmt.Errorf("parsing base template: %w", err)
	}

	// Load layout templates from theme/layouts/
	layoutsDir := filepath.Join(themeDir, "layouts")
	if info, err := os.Stat(layoutsDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(layoutsDir)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".html")
			content, err := os.ReadFile(filepath.Join(layoutsDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading layout %q: %w", name, err)
			}

			// Clone base template and parse layout on top
			tmpl, err := template.Must(tc.base.Clone()).Funcs(tc.funcMap).Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("parsing layout %q: %w", name, err)
			}
			tc.layouts[name] = tmpl
		}
	}

	// Load per-type renderer templates from theme/renderers/
	if err := tc.loadStandaloneDir(filepath.Join(themeDir, "renderers"), "renderer", tc.renderers); err != nil {
		return nil, err
	}

	// Load partial templates from theme/partials/, callable via {{partial "name" .}}
	if err := tc.loadStandaloneDir(filepath.Join(themeDir, "partials"), "partial", tc.partials); err != nil {
		return nil, err
	}

	// Detect which templates reference .JSON so the render phase only
	// serializes the payload for pages whose layout actually embeds it.
	tc.usesJSON = make(map[*template.Template]bool, len(tc.layouts)+1)
	tc.usesJSON[tc.base] = templateReferencesJSON(tc.base)
	for _, tmpl := range tc.layouts {
		tc.usesJSON[tmpl] = templateReferencesJSON(tmpl)
	}

	return tc, nil
}

// UsesJSON reports whether the given (already resolved) template references
// .JSON anywhere in its tree.
func (tc *TemplateCache) UsesJSON(tmpl *template.Template) bool {
	return tc.usesJSON[tmpl]
}

// loadStandaloneDir parses every .html file in dir as an independent template
// with the shared funcMap, storing it in dst under its base name.
func (tc *TemplateCache) loadStandaloneDir(dir, kind string, dst map[string]*template.Template) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".html")
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("reading %s %q: %w", kind, name, err)
		}

		tmpl, err := template.New(name).Funcs(tc.funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parsing %s %q: %w", kind, name, err)
		}
		dst[name] = tmpl
	}
	return nil
}

// GetLayout returns the template for the given layout name.
// Falls back to the base template if no matching layout is found.
func (tc *TemplateCache) GetLayout(name string) *template.Template {
	if name != "" {
		if tmpl, ok := tc.layouts[name]; ok {
			return tmpl
		}
	}
	return tc.base
}
