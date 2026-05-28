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

// ParseAll processes all content files in parallel, splitting frontmatter
// and dispatching to the appropriate parser. Files with no matching parser
// are passed through as raw content nodes.
func ParseAll(ctx context.Context, files []discover.FileEntry, parsers map[string]core.Parser) ([]*core.Page, []core.EngineError) {
	pages := make([]*core.Page, len(files))
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
	var validPages []*core.Page
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

	page := &core.Page{
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
