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
					// Restore the whole file from its one content-hash entry: a
					// single page for an ordinary file, root+children for a tree.
					pages[idx] = pagesFromCache(file, cp)
					cached[idx] = cp
					reused[idx] = true
					return nil
				}
			}

			built, engErr := buildPages(file, parsers, raw)
			if engErr != nil {
				errs[idx] = *engErr
				hasErr[idx] = true
				return nil
			}
			pages[idx] = built
			// Snapshot by deep copy: hooks mutate live pages in place after
			// PARSE, and the cache must store post-PARSE state only. All pages of
			// a tree are stored under this ONE file's content-hash entry.
			cached[idx] = buildcache.ClonePage(cacheEntry(hash, built))
			return nil
		})
	}

	g.Wait()

	var validPages []*core.Page
	var validErrs []core.EngineError
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
		newMap[files[i].RelPath] = cached[i]
	}

	return validPages, validErrs, newMap, stats
}

// cacheEntry builds the ParsedPage that stores a file's whole output under its
// single content-hash entry. built[0] is the root (an ordinary page or a tree
// root); built[1:] are the tree's children in PARSE's deterministic slug-path
// order, each stored as a TreeNode.
func cacheEntry(hash string, built []*core.Page) buildcache.ParsedPage {
	root := built[0]
	entry := buildcache.ParsedPage{
		ContentHash: hash,
		Envelope:    root.Envelope,
		Nodes:       root.Nodes,
		Type:        root.Type,
		Layout:      root.Layout,
		IsRaw:       root.IsRaw,
	}
	if len(built) > 1 {
		entry.Tree = make([]buildcache.TreeNode, 0, len(built)-1)
		for _, child := range built[1:] {
			entry.Tree = append(entry.Tree, buildcache.TreeNode{
				TreeSlugPath: child.TreeSlugPath,
				Envelope:     child.Envelope,
				Nodes:        child.Nodes,
				Type:         child.Type,
				Layout:       child.Layout,
			})
		}
	}
	return entry
}

// pagesFromCache reconstructs a file's whole post-PARSE output from its single
// content-hash entry: the root page for an ordinary file, or the root plus one
// real page per cached tree child, exactly as PARSE originally expanded them.
// The restored pages get deep copies of the cached envelope/nodes so hooks
// mutating them cannot corrupt the cache entry (which is re-persisted at the
// end of the build). The deep-copy isolation applies to every page of the tree.
func pagesFromCache(file discover.FileEntry, cp buildcache.ParsedPage) []*core.Page {
	cp = buildcache.ClonePage(cp)
	root := &core.Page{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
		Envelope:   cp.Envelope,
		Nodes:      cp.Nodes,
		Type:       cp.Type,
		Layout:     cp.Layout,
		IsRaw:      cp.IsRaw,
	}
	pages := []*core.Page{root}
	for _, tn := range cp.Tree {
		// Reconstruct a tree child exactly as buildTreePages did: its RelPath is
		// the source file's RelPath joined with its slug path; its SourcePath is
		// the artifact; TreeRootRel/TreeSlugPath let ROUTE re-compose its URL
		// under the root and let RENDER narrow the whole tree on an artifact edit.
		pages = append(pages, &core.Page{
			SourcePath:   file.Path,
			RelPath:      path.Join(filepathToSlash(file.RelPath), tn.TreeSlugPath),
			Ext:          file.Ext,
			Envelope:     tn.Envelope,
			Nodes:        tn.Nodes,
			Type:         tn.Type,
			Layout:       tn.Layout,
			TreeRootRel:  file.RelPath,
			TreeSlugPath: tn.TreeSlugPath,
		})
	}
	return pages
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
	built, engErr := buildPages(file, parsers, raw)
	if engErr != nil {
		return nil, engErr
	}
	return built[0], nil
}

// buildPages parses already-read file bytes into post-PARSE pages. Ordinary
// parsers yield a single page; a parser also implementing core.TreeParser
// yields the routed root page (built[0]) plus one real page per tree node
// (built[1:]), which the caller stores together under the file's single
// content-hash cache entry.
func buildPages(file discover.FileEntry, parsers map[string]core.Parser, raw []byte) (built []*core.Page, _ *core.EngineError) {
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
				return nil, engErr
			}
			return pages, nil
		}

		envelope, nodes, err := parser.Parse(raw)
		if err != nil {
			return nil, &core.EngineError{
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
			return nil, &core.EngineError{
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

	return []*core.Page{page}, nil
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
