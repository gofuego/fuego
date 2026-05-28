package render

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/FabioSol/fuego/core"
)

// TemplateCache holds all pre-parsed templates for rendering.
type TemplateCache struct {
	base      *template.Template
	layouts   map[string]*template.Template
	renderers map[string]*template.Template
	funcMap   template.FuncMap
}

// LoadTemplates scans the theme directory and pre-parses all templates.
// It builds the template cache used by the render phase.
func LoadTemplates(themeDir string) (*TemplateCache, error) {
	tc := &TemplateCache{
		layouts:   make(map[string]*template.Template),
		renderers: make(map[string]*template.Template),
	}

	// Build funcMap with the recursive render function
	tc.funcMap = template.FuncMap{
		"render": func(nodes []core.Node) template.HTML {
			return tc.renderWithOverrides(nodes)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}

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
	renderersDir := filepath.Join(themeDir, "renderers")
	if info, err := os.Stat(renderersDir); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(renderersDir)
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".html")
			content, err := os.ReadFile(filepath.Join(renderersDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading renderer %q: %w", name, err)
			}

			tmpl, err := template.New(name).Funcs(tc.funcMap).Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("parsing renderer %q: %w", name, err)
			}
			tc.renderers[name] = tmpl
		}
	}

	return tc, nil
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

// renderWithOverrides renders nodes using per-type renderer templates
// when available, falling back to the default semantic renderer.
func (tc *TemplateCache) renderWithOverrides(nodes []core.Node) template.HTML {
	var sb strings.Builder
	for _, n := range nodes {
		if tmpl, ok := tc.renderers[n.Type]; ok {
			// Per-type renderer exists
			tmpl.Execute(&sb, n)
		} else {
			// Default semantic renderer
			var nodeSB strings.Builder
			renderNode(&nodeSB, n)
			sb.WriteString(nodeSB.String())
		}
	}
	return template.HTML(sb.String())
}
