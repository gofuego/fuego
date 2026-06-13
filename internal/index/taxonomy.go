package index

import (
	"fmt"
	"sort"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
)

// BuildTaxonomies scans page envelopes for taxonomy fields declared in config,
// builds inverted indexes, and generates virtual pages for each taxonomy:
//   - Term pages: one per unique term value (e.g., /tags/go/)
//   - Index pages: one per taxonomy listing all terms (e.g., /tags/)
func BuildTaxonomies(pages []*core.Page, taxonomies map[string]config.TaxonomyConfig) []*core.Page {
	var virtual []*core.Page

	// Process taxonomies in sorted order for deterministic output
	taxNames := sortedKeys(taxonomies)

	for _, fieldName := range taxNames {
		taxCfg := taxonomies[fieldName]

		// Build inverted index: term → list of pages
		termIndex := buildTermIndex(pages, fieldName)
		if len(termIndex) == 0 {
			continue
		}

		// Generate term pages, split by page_size when configured
		terms := sortedKeys(termIndex)
		for _, term := range terms {
			memberPages := termIndex[term]
			termPage := buildTermPage(fieldName, term, memberPages, taxCfg)
			virtual = append(virtual, paginate(termPage, taxCfg.PageSize)...)
		}

		// Generate index page (listing all terms)
		if taxCfg.IndexPath != "" {
			indexPage := buildTaxonomyIndexPage(fieldName, terms, termIndex, taxCfg)
			virtual = append(virtual, indexPage)
		}
	}

	return virtual
}

// buildTermIndex scans all pages and builds {term → []*Page} for a given field.
func buildTermIndex(pages []*core.Page, fieldName string) map[string][]*core.Page {
	index := make(map[string][]*core.Page)

	for _, page := range pages {
		terms := extractTerms(page.Envelope, fieldName)
		for _, term := range terms {
			normalized := normalizeTerm(term)
			if normalized != "" {
				index[normalized] = append(index[normalized], page)
			}
		}
	}

	return index
}

// extractTerms gets taxonomy values from an envelope field.
// Supports both single string and list of strings.
func extractTerms(env core.Envelope, fieldName string) []string {
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

// normalizeTerm lowercases and trims whitespace from a taxonomy term.
func normalizeTerm(term string) string {
	return strings.ToLower(strings.TrimSpace(term))
}

// buildTermPage creates a virtual page for a single taxonomy term.
func buildTermPage(fieldName, term string, members []*core.Page, cfg config.TaxonomyConfig) *core.Page {
	// Build page-ref nodes for each member
	var nodes []core.Node
	for _, member := range members {
		title, _ := member.Envelope["title"].(string)
		nodes = append(nodes, core.Node{
			Type: "page-ref",
			Attributes: map[string]any{
				"url":   member.URL,
				"title": title,
				"type":  member.Type,
			},
		})
	}

	url := expandTaxonomyPath(cfg.Path, term)

	return &core.Page{
		RelPath: fmt.Sprintf("_virtual/taxonomy/%s/%s", fieldName, term),
		Envelope: core.Envelope{
			"title":    fmt.Sprintf("%s: %s", capitalize(fieldName), term),
			"taxonomy": fieldName,
			"term":     term,
		},
		Nodes:  nodes,
		URL:    url,
		Layout: cfg.Layout,
		Type:   "taxonomy-term",
	}
}

// buildTaxonomyIndexPage creates a virtual page listing all terms for a taxonomy.
func buildTaxonomyIndexPage(fieldName string, terms []string, termIndex map[string][]*core.Page, cfg config.TaxonomyConfig) *core.Page {
	var nodes []core.Node
	for _, term := range terms {
		termURL := expandTaxonomyPath(cfg.Path, term)
		nodes = append(nodes, core.Node{
			Type: "term-ref",
			Attributes: map[string]any{
				"term":  term,
				"count": len(termIndex[term]),
				"url":   termURL,
			},
		})
	}

	url := cfg.IndexPath
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	return &core.Page{
		RelPath: fmt.Sprintf("_virtual/taxonomy/%s/_index", fieldName),
		Envelope: core.Envelope{
			"title":    fmt.Sprintf("All %s", capitalize(fieldName)),
			"taxonomy": fieldName,
		},
		Nodes:  nodes,
		URL:    url,
		Layout: cfg.IndexLayout,
		Type:   "taxonomy-index",
	}
}

// expandTaxonomyPath replaces {term} in a path pattern and normalizes slashes.
func expandTaxonomyPath(pattern, term string) string {
	result := strings.ReplaceAll(pattern, "{term}", term)
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}
	if !strings.HasSuffix(result, "/") {
		result = result + "/"
	}
	return result
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
