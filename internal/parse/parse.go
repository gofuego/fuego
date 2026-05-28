package parse

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/discover"
	"golang.org/x/sync/errgroup"
)

// PageData is the central data carrier flowing through the pipeline.
// It is progressively enriched by each phase: DISCOVER sets paths,
// PARSE sets envelope/nodes, ROUTE sets URL, INDEX may annotate further.
type PageData struct {
	SourcePath string          // absolute path to the source file
	RelPath    string          // path relative to content dir
	Ext        string          // file extension without dot
	Envelope   core.Envelope // parsed frontmatter metadata
	Nodes      []core.Node   // parsed AST body
	URL        string          // resolved output URL (set by route phase)
	Layout     string          // layout name from envelope
	Type       string          // content type (from envelope or extension)
	IsRaw      bool            // true if no parser matched (raw passthrough)
}

// ParseAll processes all content files in parallel, splitting frontmatter
// and dispatching to the appropriate parser. Files with no matching parser
// are passed through as raw content nodes.
func ParseAll(ctx context.Context, files []discover.FileEntry, parsers map[string]core.Parser) ([]*PageData, []core.EngineError) {
	pages := make([]*PageData, len(files))
	errs := make([]core.EngineError, len(files))
	hasErr := make([]bool, len(files))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	for i, f := range files {
		idx := i
		file := f

		g.Go(func() error {
			page, engErr := parseFile(file, parsers)
			if engErr != nil {
				errs[idx] = *engErr
				hasErr[idx] = true
			} else {
				pages[idx] = page
			}
			return nil // never return error; we collect EngineErrors instead
		})
	}

	g.Wait()

	// Compact results
	var validPages []*PageData
	var validErrs []core.EngineError

	for i := range files {
		if hasErr[i] {
			validErrs = append(validErrs, errs[i])
		} else if pages[i] != nil {
			validPages = append(validPages, pages[i])
		}
	}

	return validPages, validErrs
}

func parseFile(file discover.FileEntry, parsers map[string]core.Parser) (*PageData, *core.EngineError) {
	raw, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, &core.EngineError{
			Phase:    "PARSE",
			File:     file.RelPath,
			Severity: core.LocalFatal,
			Err:      fmt.Errorf("reading file: %w", err),
		}
	}

	envelope, payload, err := SplitFrontmatter(raw)
	if err != nil {
		return nil, &core.EngineError{
			Phase:    "PARSE",
			File:     file.RelPath,
			Severity: core.LocalFatal,
			Err:      err,
		}
	}
	if envelope == nil {
		envelope = make(core.Envelope)
	}

	page := &PageData{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
		Envelope:   envelope,
	}

	// Determine type: frontmatter "type" field overrides extension
	if t, ok := envelope["type"].(string); ok && t != "" {
		page.Type = t
	} else {
		page.Type = file.Ext
	}

	// Resolve layout from frontmatter
	if l, ok := envelope["layout"].(string); ok && l != "" {
		page.Layout = l
	}

	// Find parser: first by Type, then by extension
	parser, found := parsers[page.Type]
	if !found && page.Type != file.Ext {
		parser, found = parsers[file.Ext]
	}

	if found {
		nodes, err := parser.Parse(payload, envelope)
		if err != nil {
			return nil, &core.EngineError{
				Phase:    "PARSE",
				File:     file.RelPath,
				Severity: core.LocalFatal,
				Err:      fmt.Errorf("parser %q: %w", page.Type, err),
			}
		}
		page.Nodes = nodes
	} else {
		// Raw passthrough: single node with the entire payload as content
		page.IsRaw = true
		content := strings.TrimSpace(string(payload))
		if content != "" {
			page.Nodes = []core.Node{
				{Type: "raw", Content: content},
			}
		}
	}

	return page, nil
}
