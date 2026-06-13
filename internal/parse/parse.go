package parse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/discover"
	"golang.org/x/sync/errgroup"
)

// ParseAll processes all content files in parallel, dispatching to the
// appropriate parser. Files with no matching parser are passed through
// as raw content nodes.
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

	page := &core.Page{
		SourcePath: file.Path,
		RelPath:    file.RelPath,
		Ext:        file.Ext,
	}

	// Parser dispatch: by extension, then by filename match
	page.Type = file.Ext

	parser, found := parsers[file.Ext]
	if !found && file.MatchedParser != "" {
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
				{Type: "raw", Content: content},
			}
		}
	}

	return page, nil
}
