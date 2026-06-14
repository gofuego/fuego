package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// PageEntry is a single page in the manifest.
type PageEntry struct {
	URL      string         `json:"url"`
	Type     string         `json:"type"`
	Layout   string         `json:"layout,omitempty"`
	Title    string         `json:"title,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Envelope map[string]any `json:"envelope,omitempty"`
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

		entries[i] = PageEntry{
			URL:      p.URL,
			Type:     p.Type,
			Layout:   p.Layout,
			Title:    title,
			Summary:  summary,
			Envelope: env,
		}
	}

	m := &Manifest{
		Pages: entries,
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
			if page.Type == "taxonomy-term" || page.Type == "taxonomy-index" || page.Type == "collection" {
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
