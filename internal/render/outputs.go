package render

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	texttemplate "text/template"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// OutputData is the data passed to site-level output templates.
type OutputData struct {
	Site SiteTemplateData
}

// RenderOutputs executes every file under theme/outputs/ (user theme and
// pack themes, user wins) as a text/template fed with .Site, writing the
// result to the same relative path in the output directory. text/template
// is deliberate: outputs are XML/JSON/plain text, where html/template
// escaping would corrupt the result.
func RenderOutputs(pages []*core.Page, cfg *config.Config, packs []core.Pack) []core.EngineError {
	layers := themeLayers(cfg.Dirs.Theme, packs)

	files := collectOutputFiles(layers)
	if len(files) == 0 {
		return nil
	}

	data := OutputData{Site: SiteTemplateData{
		Name:    cfg.Site.Name,
		BaseURL: cfg.Site.BaseURL,
		Pages:   BuildPageRefs(pages),
	}}

	// Page output paths, for collision detection against output files.
	pagePaths := make(map[string]string, len(pages))
	for _, p := range pages {
		rel := strings.TrimPrefix(strings.TrimSuffix(p.URL, "/")+"/index.html", "/")
		pagePaths[rel] = p.RelPath
	}

	rels := make([]string, 0, len(files))
	for rel := range files {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	var errs []core.EngineError
	for _, rel := range rels {
		if claimant, ok := pagePaths[rel]; ok {
			errs = append(errs, core.EngineError{
				Phase:    "OUTPUTS",
				File:     path.Join("outputs", rel),
				Severity: core.GlobalFatal,
				Err: fmt.Errorf("output file collides with page %q (both write %s)",
					claimant, rel),
			})
			continue
		}

		if err := renderOutputFile(rel, files[rel], data, cfg.Dirs.Output); err != nil {
			errs = append(errs, core.EngineError{
				Phase:    "OUTPUTS",
				File:     path.Join("outputs", rel),
				Severity: core.LocalFatal,
				Err:      err,
			})
		}
	}
	return errs
}

func renderOutputFile(rel, content string, data OutputData, outputDir string) error {
	tmpl, err := texttemplate.New(rel).Funcs(textFuncMap()).Parse(content)
	if err != nil {
		return fmt.Errorf("parsing output template: %w", err)
	}

	outPath := filepath.Join(outputDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing output template: %w", err)
	}
	return nil
}

// collectOutputFiles walks outputs/ in every theme layer, including nested
// directories and any file extension. Later layers override earlier ones;
// pack-vs-pack overrides warn, the user theme overriding a pack is silent.
func collectOutputFiles(layers []themeLayer) map[string]string {
	files := make(map[string]string)
	owner := make(map[string]string)

	for _, layer := range layers {
		fs.WalkDir(layer.fsys, "outputs", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // a layer without outputs/ is fine
			}
			rel := strings.TrimPrefix(p, "outputs/")
			content, readErr := fs.ReadFile(layer.fsys, p)
			if readErr != nil {
				return nil
			}
			if prev, taken := owner[rel]; taken && prev != "" && layer.packName != "" {
				fmt.Fprintf(os.Stderr, "fuego: warning: outputs/%s from pack %q overridden by pack %q\n",
					rel, prev, layer.packName)
			}
			files[rel] = string(content)
			owner[rel] = layer.packName
			return nil
		})
	}
	return files
}

// textFuncMap exposes the data-shaping template funcs to text/template
// outputs. The HTML-specific funcs (partial, render, safeHTML) are omitted:
// they produce html/template values that don't belong in XML/JSON output.
func textFuncMap() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"dict":       dictFunc,
		"where":      whereFunc,
		"sortBy":     sortByFunc,
		"limit":      limitFunc,
		"first":      firstFunc,
		"dateFormat": dateFormatFunc,
	}
}
