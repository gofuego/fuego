package route

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// ResolveAll assigns a URL to each page using the three-tier priority:
//  1. Frontmatter "slug" field (leaf segment override)
//  2. Config route pattern for the page's type
//  3. Filesystem mirror (default)
//
// Pages expanded from a TreeParser (TreeSlugPath set) are resolved in a second
// pass: their URL is the tree root's resolved URL joined with the child's
// slug-path segments, so the root's three-tier routing — including the
// index-file convention — is composed first and the whole tree hangs beneath
// it. After resolution, it checks for URL collisions.
func ResolveAll(pages []*core.Page, cfg *config.Config) []core.EngineError {
	// Pass 1: resolve every ordinary page (including tree roots) so a root's
	// URL exists before its children compose under it.
	rootURLByRel := make(map[string]string)
	for _, page := range pages {
		if page.TreeSlugPath != "" {
			continue
		}
		page.URL = resolveURL(page, cfg)
		rootURLByRel[page.RelPath] = page.URL
	}

	// Pass 2: compose tree-child URLs under their root's resolved URL.
	for _, page := range pages {
		if page.TreeSlugPath == "" {
			continue
		}
		page.URL = composeChildURL(rootURLByRel[page.TreeRootRel], page.TreeSlugPath)
	}

	return DetectCollisions(pages)
}

// composeChildURL joins a tree child's slug path under the root's resolved URL,
// producing a normalized, trailing-slashed URL. An empty rootURL (no root page
// resolved, which should not happen for a well-formed tree) falls back to a
// root-relative path so the child still routes and any collision is visible.
func composeChildURL(rootURL, slugPath string) string {
	base := strings.TrimSuffix(rootURL, "/")
	url := base + "/" + strings.Trim(slugPath, "/") + "/"
	if !strings.HasPrefix(url, "/") {
		url = "/" + url
	}
	for strings.Contains(url, "//") {
		url = strings.ReplaceAll(url, "//", "/")
	}
	return url
}

func resolveURL(page *core.Page, cfg *config.Config) string {
	relPath := page.RelPath
	ext := filepath.Ext(relPath)
	filename := strings.TrimSuffix(filepath.Base(relPath), ext)
	dir := filepath.ToSlash(filepath.Dir(relPath))
	if dir == "." {
		dir = ""
	}

	// Determine slug: frontmatter slug overrides filename
	slug := filename
	if s, ok := page.Envelope["slug"].(string); ok && s != "" {
		slug = s
	}

	// Three-tier priority:
	// 1. Config route pattern for the page's content type (with slug override applied)
	// 2. Filesystem mirror (with slug override applied)
	if pattern, ok := cfg.Routes[page.Type]; ok {
		return expandPattern(pattern, dir, slug, filename)
	}

	// Filesystem mirror fallback. An "index" file is the root of its directory:
	// content/index.md → "/", content/blog/index.md → "/blog/".
	if slug == "index" {
		if dir == "" {
			return "/"
		}
		return "/" + dir + "/"
	}
	if dir == "" {
		return "/" + slug + "/"
	}
	return "/" + dir + "/" + slug + "/"
}

// expandPattern replaces {dir}, {slug}, and {filename} placeholders in a route pattern.
// The result is cleaned to avoid double slashes and always ends with "/".
func expandPattern(pattern, dir, slug, filename string) string {
	result := pattern
	result = strings.ReplaceAll(result, "{dir}", dir)
	result = strings.ReplaceAll(result, "{slug}", slug)
	result = strings.ReplaceAll(result, "{filename}", filename)

	// Clean up: collapse double slashes from empty {dir}
	for strings.Contains(result, "//") {
		result = strings.ReplaceAll(result, "//", "/")
	}

	// Ensure leading slash
	if !strings.HasPrefix(result, "/") {
		result = "/" + result
	}

	// Ensure trailing slash
	if !strings.HasSuffix(result, "/") {
		result = result + "/"
	}

	return result
}

// DetectCollisions checks for duplicate URLs across all pages.
// URLs are compared by output identity: "/overview" and "/overview/" both
// write overview/index.html, so they collide regardless of trailing slash.
// Returns GlobalFatal errors for each collision pair.
func DetectCollisions(pages []*core.Page) []core.EngineError {
	seen := make(map[string]*core.Page)
	var errs []core.EngineError

	for _, page := range pages {
		key := strings.TrimSuffix(page.URL, "/")
		if existing, ok := seen[key]; ok {
			errs = append(errs, core.EngineError{
				Phase:    "ROUTE",
				File:     page.RelPath,
				Severity: core.GlobalFatal,
				Err: fmt.Errorf("URL collision: %q is claimed by both %q and %q",
					page.URL, existing.RelPath, page.RelPath),
			})
		} else {
			seen[key] = page
		}
	}

	return errs
}
