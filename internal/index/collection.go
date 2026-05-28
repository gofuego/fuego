package index

import (
	"fmt"
	"sort"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/parse"
	"github.com/bmatcuk/doublestar/v4"
)

// BuildCollections creates virtual collection pages by matching pages against
// glob patterns and sorting by a configured envelope field.
func BuildCollections(pages []*parse.PageData, collections map[string]config.CollectionConfig) []*parse.PageData {
	var virtual []*parse.PageData

	// Process collections in sorted order for deterministic output
	colNames := sortedKeys(collections)

	for _, name := range colNames {
		colCfg := collections[name]
		colPage := buildCollectionPage(name, pages, colCfg)
		if colPage != nil {
			virtual = append(virtual, colPage)
		}
	}

	return virtual
}

func buildCollectionPage(name string, pages []*parse.PageData, cfg config.CollectionConfig) *parse.PageData {
	// Match pages by glob pattern on relative path
	var members []*parse.PageData
	for _, page := range pages {
		normalized := strings.ReplaceAll(page.RelPath, "\\", "/")
		matched, err := doublestar.Match(cfg.Match, normalized)
		if err != nil {
			continue
		}
		if matched {
			members = append(members, page)
		}
	}

	if len(members) == 0 {
		return nil
	}

	// Sort by envelope field
	if cfg.SortBy != "" {
		sortByField(members, cfg.SortBy)
	}

	// Build page-ref nodes
	var nodes []core.Node
	for _, member := range members {
		title, _ := member.Envelope["title"].(string)
		attrs := map[string]any{
			"url":   member.URL,
			"title": title,
			"type":  member.Type,
		}
		// Include the sort field value in attributes if present
		if cfg.SortBy != "" {
			if val, ok := member.Envelope[cfg.SortBy]; ok {
				attrs[cfg.SortBy] = val
			}
		}
		nodes = append(nodes, core.Node{
			Type:       "page-ref",
			Attributes: attrs,
		})
	}

	url := cfg.Path
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	if !strings.HasSuffix(url, "/") {
		url = url + "/"
	}

	return &parse.PageData{
		RelPath: fmt.Sprintf("_virtual/collection/%s", name),
		Envelope: core.Envelope{
			"title":      fmt.Sprintf("%s", capitalize(name)),
			"collection": name,
		},
		Nodes:  nodes,
		URL:    url,
		Layout: cfg.Layout,
		Type:   "collection",
	}
}

// sortByField sorts pages by an envelope field value.
// Supports string and numeric comparisons.
func sortByField(pages []*parse.PageData, field string) {
	sort.SliceStable(pages, func(i, j int) bool {
		vi := pages[i].Envelope[field]
		vj := pages[j].Envelope[field]

		// Try numeric comparison first
		ni, niOk := toFloat64(vi)
		nj, njOk := toFloat64(vj)
		if niOk && njOk {
			return ni < nj
		}

		// Fall back to string comparison
		si := fmt.Sprintf("%v", vi)
		sj := fmt.Sprintf("%v", vj)
		return si < sj
	})
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case float64:
		return n, true
	case int64:
		return float64(n), true
	}
	return 0, false
}
