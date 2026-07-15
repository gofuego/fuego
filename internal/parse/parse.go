package parse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/buildcache"
	"github.com/gofuego/fuego/internal/discover"
	"golang.org/x/sync/errgroup"
)

// CacheStats reports what an incremental parse reused versus reparsed.
type CacheStats struct {
	Parsed  int             // files parsed this build
	Reused  int             // files served from cache
	Changed map[string]bool // RelPaths parsed this build (new or modified)
}

// ParseAll processes all content files in parallel, dispatching to the
// appropriate parser. Files with no matching parser are passed through
// as raw content nodes.
func ParseAll(ctx context.Context, files []discover.FileEntry, parsers map[string]core.Parser) ([]*core.Page, []core.EngineError) {
	pages, errs, _, _ := ParseAllCached(ctx, files, parsers, nil)
	return pages, errs
}

// ParseAllCached is like ParseAll but reuses parsed pages from prev whose
// content hash is unchanged, parsing only new or modified files. It returns
// the compacted pages and errors, the fresh page map to persist (covering
// exactly the current files), and reuse statistics. prev may be nil.
func ParseAllCached(ctx context.Context, files []discover.FileEntry, parsers map[string]core.Parser, prev map[string]buildcache.ParsedPage) ([]*core.Page, []core.EngineError, map[string]buildcache.ParsedPage, CacheStats) {
	// pages[idx] holds every page produced from file idx: one entry for an
	// ordinary file, a root-plus-children slice for a TreeParser file.
	pages := make([][]*core.Page, len(files))
	errs := make([]core.EngineError, len(files))
	hasErr := make([]bool, len(files))
	cached := make([]buildcache.ParsedPage, len(files))
	reused := make([]bool, len(files))
	// isTree[idx] marks files expanded by a TreeParser. Their multi-page output
	// has no single-entry cache representation yet (issue 04), so they are
	// excluded from the cache — a per-page degradation, warned — keeping
	// clean/incremental byte-equivalence green.
	isTree := make([]bool, len(files))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for i, f := range files {
		idx := i
		file := f

		g.Go(func() error {
			raw, err := os.ReadFile(file.Path)
			if err != nil {
				errs[idx] = core.EngineError{Phase: "PARSE", File: file.RelPath, Severity: core.LocalFatal, Err: fmt.Errorf("reading file: %w", err)}
				hasErr[idx] = true
				return nil
			}
			hash := buildcache.HashBytes(raw)

			if prev != nil {
				if cp, ok := prev[file.RelPath]; ok && cp.ContentHash == hash {
					pages[idx] = []*core.Page{pageFromCache(file, cp)}
					cached[idx] = cp
					reused[idx] = true
					return nil
				}
			}

			built, tree, engErr := buildPages(file, parsers, raw)
			if engErr != nil {
				errs[idx] = *engErr
				hasErr[idx] = true
				return nil
			}
			pages[idx] = built
			isTree[idx] = tree
			if tree {
				// Not cacheable in this slice; leave cached[idx] zero and skip
				// storing it in newMap below.
				return nil
			}
			root := built[0]
			// Snapshot by deep copy: hooks mutate live pages in place after
			// PARSE, and the cache must store post-PARSE state only.
			cached[idx] = buildcache.ClonePage(buildcache.ParsedPage{
				ContentHash: hash,
				Envelope:    root.Envelope,
				Nodes:       root.Nodes,
				Type:        root.Type,
				Layout:      root.Layout,
				IsRaw:       root.IsRaw,
			})
			return nil
		})
	}

	g.Wait()

	var validPages []*core.Page
	var validErrs []core.EngineError
	var treeUncached []string
	newMap := make(map[string]buildcache.ParsedPage, len(files))
	stats := CacheStats{Changed: map[string]bool{}}

	for i := range files {
		if hasErr[i] {
			validErrs = append(validErrs, errs[i])
			continue
		}
		if len(pages[i]) == 0 {
			continue
		}
		validPages = append(validPages, pages[i]...)
		if reused[i] {
			stats.Reused++
			newMap[files[i].RelPath] = cached[i]
			continue
		}
		stats.Parsed++
		stats.Changed[files[i].RelPath] = true
		if isTree[i] {
			// Excluded from the cache; every build reparses it (see isTree).
			treeUncached = append(treeUncached, files[i].RelPath)
			continue
		}
		newMap[files[i].RelPath] = cached[i]
	}

	if len(treeUncached) > 0 {
		sort.Strings(treeUncached)
		fmt.Fprintf(os.Stderr, "fuego: warning: %d tree-parsed file(s) excluded from the build cache (reparsed every build): %s\n",
			len(treeUncached), strings.Join(treeUncached, ", "))
	}

	return validPages, validErrs, newMap, stats
}

// pageFromCache reconstructs a post-PARSE page from a cache entry, taking the
// filesystem paths from the freshly discovered file. The restored page gets a
// deep copy of the cached envelope/nodes so hooks mutating it cannot corrupt
// the cache entry (which is re-persisted at the end of the build).
func pageFromCache(file discover.FileEntry, cp buildcache.ParsedPage) *core.Page {
	cp = buildcache.ClonePage(cp)
	return &core.Page{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
		Envelope:   cp.Envelope,
		Nodes:      cp.Nodes,
		Type:       cp.Type,
		Layout:     cp.Layout,
		IsRaw:      cp.IsRaw,
	}
}

// parseErrorLine extracts the source line from a core.ParseError anywhere
// in the error chain, or 0 when the parser didn't report a position.
func parseErrorLine(err error) int {
	var pe *core.ParseError
	if errors.As(err, &pe) {
		return pe.Line
	}
	return 0
}

func parseFile(file discover.FileEntry, parsers map[string]core.Parser) (*core.Page, *core.EngineError) {
	raw, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, &core.EngineError{
			Phase:    "PARSE",
			File:     file.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("reading file: %w", err),
		}
	}
	built, _, engErr := buildPages(file, parsers, raw)
	if engErr != nil {
		return nil, engErr
	}
	return built[0], nil
}

// buildPages parses already-read file bytes into post-PARSE pages. Ordinary
// parsers yield a single page; a parser also implementing core.TreeParser
// yields the routed root page plus one real page per tree node, and reports
// isTree=true so the cache can exclude the multi-page file (issue 04).
func buildPages(file discover.FileEntry, parsers map[string]core.Parser, raw []byte) (built []*core.Page, isTree bool, _ *core.EngineError) {
	page := &core.Page{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
	}

	// Parser dispatch: use the parser the dispatch resolver assigned at
	// DISCOVER (patterns before extension, longest pattern wins, ties by
	// precedence). Type is the matched parser's Type(); Ext stays the literal
	// extension. A raw asset that slipped into contentFiles has no matched
	// parser, so Type falls back to the extension and it takes the raw path.
	page.Type = file.Ext

	var parser core.Parser
	var found bool
	if file.MatchedParser != "" {
		parser, found = parsers[file.MatchedParser]
		if found {
			page.Type = file.MatchedParser
		}
	}

	if found {
		// TreeParser detection is a plain interface assertion at PARSE: a parser
		// implementing it expands one artifact into a tree of real pages. Plain
		// Parsers are untouched (no registration change).
		if tp, ok := parser.(core.TreeParser); ok {
			pages, engErr := buildTreePages(file, tp, raw, page.Type)
			if engErr != nil {
				return nil, false, engErr
			}
			return pages, true, nil
		}

		envelope, nodes, err := parser.Parse(raw)
		if err != nil {
			return nil, false, &core.EngineError{
				Phase:    "PARSE",
				File:     file.RelPath,
				Line:     parseErrorLine(err),
				Severity: core.LocalFatal,
				Err:      fmt.Errorf("parser %q: %w", page.Type, err),
			}
		}
		if envelope == nil {
			envelope = make(core.Envelope)
		}
		page.Envelope = envelope
		page.Nodes = nodes

		// Resolve layout from envelope
		if l, ok := envelope["layout"].(string); ok && l != "" {
			page.Layout = l
		}

		// Allow envelope type to set Page.Type for template use
		if t, ok := envelope["type"].(string); ok && t != "" {
			page.Type = t
		}
	} else {
		// Raw passthrough: split frontmatter for metadata, use payload as content
		envelope, payload, fmErr := core.SplitFrontmatter(raw)
		if fmErr != nil {
			return nil, false, &core.EngineError{
				Phase:    "PARSE",
				File:     file.RelPath,
				Line:     parseErrorLine(fmErr),
				Severity: core.LocalFatal,
				Err:      fmErr,
			}
		}
		if envelope == nil {
			envelope = make(core.Envelope)
		}
		page.Envelope = envelope
		page.IsRaw = true

		// Resolve layout from envelope
		if l, ok := envelope["layout"].(string); ok && l != "" {
			page.Layout = l
		}

		content := strings.TrimSpace(string(payload))
		if content != "" {
			page.Nodes = []core.Node{
				{Type: "raw", Content: content, Raw: true},
			}
		}
	}

	return []*core.Page{page}, false, nil
}

// buildTreePages calls a TreeParser and flattens the returned PageTree into
// the routed root page plus one real core.Page per tree node. The root carries
// the tree's own Envelope/Nodes; each child's RelPath is the source file's
// RelPath joined with its slug path, and its TreeRootRel/TreeSlugPath are set
// so ROUTE can compose its URL under the root's resolved URL. Children flow
// through the pipeline as ordinary pages from here on.
func buildTreePages(file discover.FileEntry, tp core.TreeParser, raw []byte, pageType string) ([]*core.Page, *core.EngineError) {
	tree, err := tp.ParseTree(raw)
	if err != nil {
		return nil, &core.EngineError{
			Phase:    "PARSE",
			File:     file.RelPath,
			Line:     parseErrorLine(err),
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("parser %q: %w", pageType, err),
		}
	}
	if tree == nil {
		return nil, &core.EngineError{
			Phase:    "PARSE",
			File:     file.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("parser %q: ParseTree returned a nil tree", pageType),
		}
	}

	root := &core.Page{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
		Type:       pageType,
	}
	applyTreeNode(root, tree.Envelope, tree.Nodes)

	pages := []*core.Page{root}
	// Walk children depth-first in sorted slug-path order for determinism.
	pages = appendTreeChildren(pages, file, tree.Children, "", pageType)
	return pages, nil
}

// appendTreeChildren expands one level of a PageTree's children (each keyed by
// a relative slug path) into real pages, recursing into nested trees. prefix is
// the slug path accumulated from ancestor keys.
func appendTreeChildren(pages []*core.Page, file discover.FileEntry, children map[string]*core.PageTree, prefix, pageType string) []*core.Page {
	for _, key := range sortedTreeKeys(children) {
		child := children[key]
		if child == nil {
			continue
		}
		slugPath := path.Join(prefix, key)
		p := &core.Page{
			// Child identity: source file's RelPath + "/" + slug path. Its
			// source is the artifact itself (manifest multi-entry mapping is
			// issue 04), so SourcePath is the artifact's path.
			SourcePath:   file.Path,
			RelPath:      path.Join(filepathToSlash(file.RelPath), slugPath),
			Ext:          file.Ext,
			Type:         pageType,
			TreeRootRel:  file.RelPath,
			TreeSlugPath: slugPath,
		}
		applyTreeNode(p, child.Envelope, child.Nodes)
		pages = append(pages, p)
		pages = appendTreeChildren(pages, file, child.Children, slugPath, pageType)
	}
	return pages
}

// applyTreeNode copies a tree node's envelope and nodes onto a page and applies
// the same envelope conventions PARSE uses for ordinary pages (layout, type).
func applyTreeNode(p *core.Page, envelope core.Envelope, nodes []core.Node) {
	if envelope == nil {
		envelope = make(core.Envelope)
	}
	p.Envelope = envelope
	p.Nodes = nodes
	if l, ok := envelope["layout"].(string); ok && l != "" {
		p.Layout = l
	}
	if t, ok := envelope["type"].(string); ok && t != "" {
		p.Type = t
	}
}

func sortedTreeKeys(m map[string]*core.PageTree) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// filepathToSlash converts an OS path to forward slashes so tree RelPaths are
// composed consistently regardless of platform.
func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
