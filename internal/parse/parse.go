package parse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
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
	pages := make([]*core.Page, len(files))
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
					pages[idx] = pageFromCache(file, cp)
					cached[idx] = cp
					reused[idx] = true
					return nil
				}
			}

			page, engErr := buildPage(file, parsers, raw)
			if engErr != nil {
				errs[idx] = *engErr
				hasErr[idx] = true
				return nil
			}
			pages[idx] = page
			// Snapshot by deep copy: hooks mutate live pages in place after
			// PARSE, and the cache must store post-PARSE state only.
			cached[idx] = buildcache.ClonePage(buildcache.ParsedPage{
				ContentHash: hash,
				Envelope:    page.Envelope,
				Nodes:       page.Nodes,
				Type:        page.Type,
				Layout:      page.Layout,
				IsRaw:       page.IsRaw,
			})
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
		if pages[i] == nil {
			continue
		}
		validPages = append(validPages, pages[i])
		newMap[files[i].RelPath] = cached[i]
		if reused[i] {
			stats.Reused++
		} else {
			stats.Parsed++
			stats.Changed[files[i].RelPath] = true
		}
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
	return buildPage(file, parsers, raw)
}

// buildPage parses already-read file bytes into a post-PARSE page.
func buildPage(file discover.FileEntry, parsers map[string]core.Parser, raw []byte) (*core.Page, *core.EngineError) {
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

	return page, nil
}
