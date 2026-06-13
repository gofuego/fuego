package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/FabioSol/fuego/core"
)

// TemplateCache holds all pre-parsed templates for rendering.
type TemplateCache struct {
	base      *template.Template
	layouts   map[string]*template.Template
	renderers map[string]*template.Template
	partials  map[string]*template.Template
	funcMap   template.FuncMap
	usesJSON  map[*template.Template]bool
	usesSite  map[*template.Template]bool
}

// themeLayer is one source of theme templates. Layers are ordered lowest
// precedence first: packs in registration order, the user theme dir last.
type themeLayer struct {
	packName string // "" for the user theme dir
	fsys     fs.FS
}

// themeLayers builds the ordered layer list: pack themes in registration
// order, then the user theme directory (highest precedence) if it exists.
func themeLayers(themeDir string, packs []core.Pack) []themeLayer {
	var layers []themeLayer
	for _, p := range packs {
		if p.Theme != nil {
			layers = append(layers, themeLayer{packName: p.Name, fsys: p.Theme})
		}
	}
	if info, err := os.Stat(themeDir); err == nil && info.IsDir() {
		layers = append(layers, themeLayer{fsys: os.DirFS(themeDir)})
	}
	return layers
}

// LoadTemplates pre-parses all templates from the user theme directory
// layered over any registered pack themes. The user's files always win;
// among packs, later registration wins with a logged warning.
func LoadTemplates(themeDir string, packs []core.Pack) (*TemplateCache, error) {
	tc := &TemplateCache{
		layouts:   make(map[string]*template.Template),
		renderers: make(map[string]*template.Template),
		partials:  make(map[string]*template.Template),
	}

	tc.funcMap = tc.buildFuncMap()

	layers := themeLayers(themeDir, packs)

	// base.html from the highest layer that provides it.
	var baseContent []byte
	found := false
	for i := len(layers) - 1; i >= 0; i-- {
		if b, err := fs.ReadFile(layers[i].fsys, "base.html"); err == nil {
			baseContent = b
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("reading base template: no base.html in %s or any registered pack theme", themeDir)
	}

	var err error
	tc.base, err = template.New("base.html").Funcs(tc.funcMap).Parse(string(baseContent))
	if err != nil {
		return nil, fmt.Errorf("parsing base template: %w", err)
	}

	layoutSrc, err := mergeLayerDir(layers, "layouts")
	if err != nil {
		return nil, err
	}
	for name, content := range layoutSrc {
		// Clone base template and parse layout on top
		tmpl, err := template.Must(tc.base.Clone()).Funcs(tc.funcMap).Parse(content)
		if err != nil {
			return nil, fmt.Errorf("parsing layout %q: %w", name, err)
		}
		tc.layouts[name] = tmpl
	}

	for kind, dst := range map[string]map[string]*template.Template{
		"renderers": tc.renderers,
		"partials":  tc.partials,
	} {
		src, err := mergeLayerDir(layers, kind)
		if err != nil {
			return nil, err
		}
		for name, content := range src {
			tmpl, err := template.New(name).Funcs(tc.funcMap).Parse(content)
			if err != nil {
				return nil, fmt.Errorf("parsing %s/%s.html: %w", kind, name, err)
			}
			dst[name] = tmpl
		}
	}

	// Detect which templates reference .JSON so the render phase only
	// serializes the payload for pages whose layout actually embeds it.
	tc.usesJSON = make(map[*template.Template]bool, len(tc.layouts)+1)
	tc.usesJSON[tc.base] = templateReferencesJSON(tc.base)
	for _, tmpl := range tc.layouts {
		tc.usesJSON[tmpl] = templateReferencesJSON(tmpl)
	}

	// Detect which templates read .Site.Pages (directly or via partials), so
	// an incremental build can skip re-rendering site-blind pages.
	baseSite, layoutSite, _ := computeSitePages(tc.base, tc.layouts, tc.partials)
	tc.usesSite = make(map[*template.Template]bool, len(tc.layouts)+1)
	tc.usesSite[tc.base] = baseSite
	for name, tmpl := range tc.layouts {
		tc.usesSite[tmpl] = baseSite || layoutSite[name]
	}

	return tc, nil
}

// UsesSitePages reports whether the given resolved template reads .Site.Pages,
// meaning its output can change when any other page is added, removed, or
// edited.
func (tc *TemplateCache) UsesSitePages(tmpl *template.Template) bool {
	return tc.usesSite[tmpl]
}

// mergeLayerDir collects {name}.html files from subDir across all layers.
// Later layers overwrite earlier ones; a pack overwriting another pack's
// template logs a warning, the user theme overriding a pack is silent.
func mergeLayerDir(layers []themeLayer, subDir string) (map[string]string, error) {
	files := make(map[string]string)
	owner := make(map[string]string)

	for _, layer := range layers {
		entries, err := fs.ReadDir(layer.fsys, subDir)
		if err != nil {
			continue // layer doesn't provide this directory
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
				continue
			}
			name := strings.TrimSuffix(entry.Name(), ".html")
			content, err := fs.ReadFile(layer.fsys, path.Join(subDir, entry.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading %s/%s.html: %w", subDir, name, err)
			}
			if prev, taken := owner[name]; taken && prev != "" && layer.packName != "" {
				fmt.Fprintf(os.Stderr, "fuego: warning: %s/%s.html from pack %q overridden by pack %q\n",
					subDir, name, prev, layer.packName)
			}
			files[name] = string(content)
			owner[name] = layer.packName
		}
	}
	return files, nil
}

// UsesJSON reports whether the given (already resolved) template references
// .JSON anywhere in its tree.
func (tc *TemplateCache) UsesJSON(tmpl *template.Template) bool {
	return tc.usesJSON[tmpl]
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
