// Package linkcheck validates internal links in the rendered output.
//
// After a build, it resolves every <a href> exactly as a browser would —
// honoring each page's <base href> and the site's base URL — and reports links
// that don't land on a generated page or copied asset. It checks the final
// HTML as shipped, so it catches dangling links and links that escape the
// deployment base URL (e.g. a root-absolute "/foo" link served under a base
// path), regardless of whether the href came from content or a template.
//
// External links (with a scheme or protocol-relative), #fragments, and anchor
// existence are intentionally not checked.
package linkcheck

import (
	"fmt"
	"html"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/buildcache"
	"github.com/gofuego/fuego/internal/config"
)

var (
	aHrefRe    = regexp.MustCompile(`(?is)<a\b[^>]*?\shref="([^"]*)"`)
	baseHrefRe = regexp.MustCompile(`(?is)<base\b[^>]*?\shref="([^"]*)"`)
	schemeRe   = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.\-]*:`)
)

// Check resolves the internal links in every rendered page and returns an
// EngineError (at the given severity) for each link that doesn't resolve to an
// existing output page or asset. It reads the built output from cfg.Dirs.Output
// and resolves against cfg.Site.BaseURL, so it must run after the output is
// fully written.
func Check(pages []*core.Page, cfg *config.Config, severity core.Severity) []core.EngineError {
	outDir := cfg.Dirs.Output
	base := "/" + strings.Trim(cfg.Site.BaseURL, "/") // "/" or "/owner/repo"
	if base == "/" {
		base = ""
	}

	targets, err := buildTargetSet(outDir, base)
	if err != nil {
		return []core.EngineError{{Phase: "LINKS", Severity: severity, Err: fmt.Errorf("scanning output for link check: %w", err)}}
	}

	var errs []core.EngineError
	for _, p := range pages {
		htmlPath := filepath.Join(outDir, filepath.FromSlash(buildcache.OutputRelPath(p.URL)))
		raw, err := os.ReadFile(htmlPath)
		if err != nil {
			continue // a page with no rendered HTML can't have broken links
		}
		doc := string(raw)

		docURL := join(base, p.URL) // the page's own served URL (a directory)
		baseHref := docURL
		if m := baseHrefRe.FindStringSubmatch(doc); m != nil {
			baseHref = html.UnescapeString(m[1])
		}

		for _, m := range aHrefRe.FindAllStringSubmatch(doc, -1) {
			href := html.UnescapeString(m[1])
			target, ok := resolve(baseHref, href)
			if !ok {
				continue // external, anchor, or unparseable — not our concern
			}
			if _, found := targets[target]; found {
				continue
			}
			src := p.RelPath
			if src == "" {
				src = "(virtual page " + p.URL + ")"
			}
			errs = append(errs, core.EngineError{
				Phase:    "LINKS",
				File:     src,
				Severity: severity,
				Err:      fmt.Errorf("broken link %q resolves to %s, which is not a generated page", m[1], target),
			})
		}
	}
	return errs
}

// buildTargetSet walks the output directory and returns the canonical served
// path of every file, plus the directory URL of every page (a dir containing
// index.html). These are the valid link targets.
func buildTargetSet(outDir, base string) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	err := filepath.WalkDir(outDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(outDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		set[canon(join(base, "/"+rel))] = struct{}{}
		if path.Base(rel) == "index.html" {
			dir := path.Dir(rel)
			if dir == "." {
				dir = ""
			}
			set[canon(join(base, "/"+dir+"/"))] = struct{}{}
		}
		return nil
	})
	if os.IsNotExist(err) {
		return nil, err
	}
	return set, err
}

// resolve resolves an href against the page's base (its <base href> or its own
// URL) the way a browser does, returning the canonical path. ok is false for
// links that aren't internal page references (external, protocol-relative,
// #fragment, or unparseable).
func resolve(base, href string) (target string, ok bool) {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "//") {
		return "", false
	}
	if schemeRe.MatchString(href) {
		return "", false // has a scheme: http:, mailto:, tel:, data:, …
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", false
	}
	r, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	return canon(b.ResolveReference(r).Path), true
}

// join concatenates a base prefix ("" or "/owner/repo") with a site URL (which
// starts with "/"), avoiding a doubled slash.
func join(base, u string) string {
	if !strings.HasPrefix(u, "/") {
		u = "/" + u
	}
	return base + u
}

// canon canonicalises a path for comparison: it collapses "//", strips a
// trailing slash, and maps the empty path to "/".
func canon(p string) string {
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	return p
}
