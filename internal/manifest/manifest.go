package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// PageEntry is a single page in the manifest.
type PageEntry struct {
	URL    string `json:"url"`
	Type   string `json:"type"`
	Layout string `json:"layout,omitempty"`
	// SourcePath is the content-dir-relative path of the source file this page
	// was built from (forward slashes). Empty for virtual pages (taxonomy,
	// collection, and other generated pages), which therefore aren't editable.
	SourcePath string `json:"source_path,omitempty"`
	// OutputPath is the generated file's path relative to the output root,
	// e.g. "blog/post/index.html" — what a host serves for this page's URL.
	OutputPath string         `json:"output_path"`
	Title      string         `json:"title,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Envelope   map[string]any `json:"envelope,omitempty"`
}

// TaxonomyEntry represents a taxonomy in the manifest.
type TaxonomyEntry struct {
	Terms map[string][]int `json:"terms"`
}

// CollectionEntry represents a collection in the manifest.
type CollectionEntry struct {
	Pages []int `json:"pages"`
}

// Manifest is the top-level site manifest structure.
type Manifest struct {
	// ContentRoot is the content directory relative to the repository (git)
	// root, e.g. "docs/content". A page's repo-relative source file is
	// ContentRoot joined with its SourcePath — what a host needs to fetch or
	// edit the source. Empty when the build is not inside a git repository.
	ContentRoot string                     `json:"content_root,omitempty"`
	Pages       []PageEntry                `json:"pages"`
	Taxonomies  map[string]TaxonomyEntry   `json:"taxonomies,omitempty"`
	Collections map[string]CollectionEntry `json:"collections,omitempty"`
}

// Generate builds a Manifest from the pipeline results.
func Generate(pages []*core.Page, cfg *config.Config) *Manifest {
	// Sort pages by URL for deterministic output
	sorted := make([]*core.Page, len(pages))
	copy(sorted, pages)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].URL < sorted[j].URL
	})

	// Build URL → index lookup
	urlIndex := make(map[string]int, len(sorted))
	entries := make([]PageEntry, len(sorted))

	for i, p := range sorted {
		urlIndex[p.URL] = i

		title, _ := p.Envelope["title"].(string)
		summary, _ := p.Envelope["summary"].(string)

		// Flatten envelope, excluding internal-only fields
		env := make(map[string]any)
		for k, v := range p.Envelope {
			env[k] = v
		}

		// Virtual pages (taxonomy/collection) have no editable source file, so
		// they carry an empty source_path — the contract a host uses to mark them
		// non-editable. Their internal "_virtual/..." RelPath stays out of the
		// manifest. See ADR-014.
		sourcePath := ""
		if !isVirtual(p) {
			sourcePath = filepath.ToSlash(p.RelPath)
		}

		entries[i] = PageEntry{
			URL:        p.URL,
			Type:       p.Type,
			Layout:     p.Layout,
			SourcePath: sourcePath,
			OutputPath: outputPath(p.URL),
			Title:      title,
			Summary:    summary,
			Envelope:   env,
		}
	}

	m := &Manifest{
		ContentRoot: contentRoot(cfg.Dirs.Content),
		Pages:       entries,
	}

	// Build taxonomy sections
	if len(cfg.Taxonomies) > 0 {
		m.Taxonomies = buildTaxonomySections(sorted, urlIndex, cfg.Taxonomies)
	}

	// Build collection sections
	if len(cfg.Collections) > 0 {
		m.Collections = buildCollectionSections(sorted, urlIndex, cfg.Collections)
	}

	return m
}

// isVirtual reports whether a page is engine-generated (a taxonomy or collection
// page) rather than backed by a source file. Virtual pages get an empty
// source_path in the manifest and are excluded from taxonomy term scanning.
func isVirtual(p *core.Page) bool {
	return p.Type == "taxonomy-term" || p.Type == "taxonomy-index" || p.Type == "collection"
}

// contentRoot returns the content directory relative to the enclosing git
// repository root (forward slashes), or "" when the dir is empty or not inside
// a git repo. This lets a host map a page's content-relative source path back to
// its real path within the repository.
func contentRoot(contentDir string) string {
	if contentDir == "" {
		return ""
	}
	abs, err := filepath.Abs(contentDir)
	if err != nil {
		return ""
	}
	root := gitRoot(abs)
	if root == "" {
		return ""
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// gitRoot walks up from start looking for a .git entry (directory or file, for
// worktrees), returning the directory that contains it, or "".
func gitRoot(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// outputPath returns the generated file path (relative to the output root) for
// a page URL, matching what the render phase writes: "<url>/index.html".
func outputPath(url string) string {
	rel := strings.Trim(url, "/")
	if rel == "" {
		return "index.html"
	}
	return rel + "/index.html"
}

// Write serializes the manifest to site-manifest.json in the output directory.
func Write(m *Manifest, outputDir string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}

	outPath := filepath.Join(outputDir, "site-manifest.json")
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

func buildTaxonomySections(pages []*core.Page, urlIndex map[string]int, taxonomies map[string]config.TaxonomyConfig) map[string]TaxonomyEntry {
	result := make(map[string]TaxonomyEntry)

	// Sorted taxonomy names for determinism
	taxNames := make([]string, 0, len(taxonomies))
	for name := range taxonomies {
		taxNames = append(taxNames, name)
	}
	sort.Strings(taxNames)

	for _, fieldName := range taxNames {
		terms := make(map[string][]int)

		for _, page := range pages {
			// Skip virtual pages — only index real content pages
			if isVirtual(page) {
				continue
			}

			pageTerms := extractTerms(page.Envelope, fieldName)
			for _, term := range pageTerms {
				idx, ok := urlIndex[page.URL]
				if ok {
					terms[term] = append(terms[term], idx)
				}
			}
		}

		if len(terms) > 0 {
			result[fieldName] = TaxonomyEntry{Terms: terms}
		}
	}

	return result
}

func buildCollectionSections(pages []*core.Page, urlIndex map[string]int, collections map[string]config.CollectionConfig) map[string]CollectionEntry {
	result := make(map[string]CollectionEntry)

	colNames := make([]string, 0, len(collections))
	for name := range collections {
		colNames = append(colNames, name)
	}
	sort.Strings(colNames)

	for _, name := range colNames {
		// Find the virtual collection page and extract member URLs from its nodes
		for _, page := range pages {
			if page.Type == "collection" {
				colName, _ := page.Envelope["collection"].(string)
				if colName != name {
					continue
				}
				var indices []int
				for _, node := range page.Nodes {
					if node.Type == "page-ref" {
						memberURL, _ := node.Attributes["url"].(string)
						if idx, ok := urlIndex[memberURL]; ok {
							indices = append(indices, idx)
						}
					}
				}
				if len(indices) > 0 {
					result[name] = CollectionEntry{Pages: indices}
				}
				break
			}
		}
	}

	return result
}

// extractTerms gets taxonomy values from an envelope field (mirrors index package logic).
func extractTerms(env map[string]any, fieldName string) []string {
	val, ok := env[fieldName]
	if !ok {
		return nil
	}

	switch v := val.(type) {
	case string:
		return []string{v}
	case []any:
		var terms []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				terms = append(terms, s)
			}
		}
		return terms
	case []string:
		return v
	}
	return nil
}
