package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/discover"
	"github.com/FabioSol/fuego/internal/index"
	"github.com/FabioSol/fuego/internal/manifest"
	"github.com/FabioSol/fuego/internal/parse"
	"github.com/FabioSol/fuego/internal/render"
	"github.com/FabioSol/fuego/internal/route"
)

// Phase identifies how far through the pipeline to run.
type Phase int

const (
	PhaseDiscover Phase = iota
	PhaseParse
	PhaseRoute
	PhaseIndex
	PhaseRender
)

// Result holds the intermediate state after a partial pipeline run.
type Result struct {
	Pages      []*core.Page
	AssetFiles []discover.FileEntry
	Errors     *core.ErrorAccumulator
}

// Build executes the full build pipeline:
// INIT → DISCOVER → PARSE → ROUTE → INDEX → RENDER → STATIC
func Build(ctx context.Context, cfg *config.Config, compiledParsers map[string]core.Parser, hooks *core.Hooks) error {
	// === PREBUILD ===
	if err := runPrebuild(cfg.Prebuild); err != nil {
		return err
	}

	// Clean the output directory
	if err := render.CleanOutput(cfg.Dirs.Output); err != nil {
		return fmt.Errorf("cleaning output directory: %w", err)
	}

	res, err := RunUntil(ctx, cfg, compiledParsers, hooks, PhaseIndex)
	if err != nil {
		return err
	}

	if res.Errors.HasFatal() {
		return reportErrors(res.Errors)
	}

	// === RENDER ===
	renderErrs := render.RenderAll(ctx, res.Pages, cfg)
	for _, e := range renderErrs {
		res.Errors.Add(e)
	}

	if res.Errors.HasFatal() {
		return reportErrors(res.Errors)
	}

	// === MANIFEST ===
	m := manifest.Generate(res.Pages, cfg)
	if err := manifest.Write(m, cfg.Dirs.Output); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// === STATIC ===
	if err := render.CopyPublicDir(cfg.Dirs.Static, cfg.Dirs.Output); err != nil {
		return fmt.Errorf("copying static files: %w", err)
	}

	if len(res.AssetFiles) > 0 {
		if err := render.CopyAssets(res.AssetFiles, cfg.Dirs.Content, cfg.Dirs.Output); err != nil {
			return fmt.Errorf("copying assets: %w", err)
		}
	}

	// Report warnings
	for _, e := range res.Errors.Errors() {
		if e.Severity == core.Warning {
			fmt.Fprintf(os.Stderr, "fuego: %s\n", e.Error())
		}
	}

	fmt.Printf("fuego: built %d pages\n", len(res.Pages))
	return nil
}

// RunUntil executes pipeline phases up to and including the given phase.
// It does NOT clean the output directory — callers that render must do that themselves.
func RunUntil(ctx context.Context, cfg *config.Config, compiledParsers map[string]core.Parser, hooks *core.Hooks, until Phase) (*Result, error) {
	acc := &core.ErrorAccumulator{}

	// === INIT ===
	// Two-tier parser merge: declarative (lowest) → compiled (highest)
	parsers := make(map[string]core.Parser)
	for name, pcfg := range cfg.Parsers {
		dp, err := parse.NewDeclarativeParser(name, pcfg)
		if err != nil {
			return nil, fmt.Errorf("initializing declarative parser: %w", err)
		}
		parsers[name] = dp
	}
	for name, p := range compiledParsers {
		parsers[name] = p
	}

	registeredTypes := make(map[string]bool)
	for name := range parsers {
		registeredTypes[name] = true
	}

	// Collect filename patterns from parsers implementing FilenameParser
	var filenamePatterns []discover.FilenamePattern
	for _, p := range parsers {
		if fp, ok := p.(core.FilenameParser); ok {
			for _, pattern := range fp.Filenames() {
				filenamePatterns = append(filenamePatterns, discover.FilenamePattern{
					Pattern:    pattern,
					ParserType: p.Type(),
				})
			}
		}
	}

	// === DISCOVER ===
	allFiles, err := discover.Walk(cfg, registeredTypes, filenamePatterns)
	if err != nil {
		return nil, fmt.Errorf("discovering content: %w", err)
	}

	var contentFiles []discover.FileEntry
	var assetFiles []discover.FileEntry
	for _, f := range allFiles {
		if f.IsAsset {
			assetFiles = append(assetFiles, f)
		} else {
			contentFiles = append(contentFiles, f)
		}
	}

	res := &Result{Errors: acc, AssetFiles: assetFiles}

	if len(contentFiles) == 0 {
		fmt.Fprintf(os.Stderr, "fuego: warning: no content files found in %s\n", cfg.Dirs.Content)
		return res, nil
	}

	if until == PhaseDiscover {
		return res, nil
	}

	// === PARSE ===
	pages, parseErrs := parse.ParseAll(ctx, contentFiles, parsers)
	for _, e := range parseErrs {
		acc.Add(e)
	}

	if acc.HasGlobalFatal() {
		return &Result{Pages: pages, AssetFiles: assetFiles, Errors: acc}, reportErrors(acc)
	}

	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "fuego: %s\n", e.Error())
		}
	}

	res.Pages = pages

	if until == PhaseParse {
		return res, nil
	}

	// === AFTER-PARSE HOOKS ===
	if hooks != nil {
		for _, hook := range hooks.AfterParse {
			pages, err = hook(pages)
			if err != nil {
				return res, fmt.Errorf("after-parse hook: %w", err)
			}
			res.Pages = pages
		}
	}

	// === ROUTE ===
	routeErrs := route.ResolveAll(pages, cfg)
	for _, e := range routeErrs {
		acc.Add(e)
	}

	if acc.HasGlobalFatal() {
		return res, reportErrors(acc)
	}

	if until == PhaseRoute {
		return res, nil
	}

	// === INDEX ===
	taxPages := index.BuildTaxonomies(pages, cfg.Taxonomies)
	colPages := index.BuildCollections(pages, cfg.Collections)
	pages = append(pages, taxPages...)
	pages = append(pages, colPages...)
	res.Pages = pages

	if len(taxPages)+len(colPages) > 0 {
		indexErrs := route.DetectCollisions(pages)
		for _, e := range indexErrs {
			acc.Add(e)
		}
		if acc.HasGlobalFatal() {
			return res, reportErrors(acc)
		}
	}

	// === BEFORE-RENDER HOOKS ===
	if hooks != nil {
		for _, hook := range hooks.BeforeRender {
			pages, err = hook(pages)
			if err != nil {
				return res, fmt.Errorf("before-render hook: %w", err)
			}
			res.Pages = pages
		}
	}

	return res, nil
}

// runPrebuild executes the prebuild shell command if configured.
func runPrebuild(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil
	}

	fmt.Printf("fuego: running prebuild: %s\n", command)
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("prebuild command failed: %w", err)
	}
	return nil
}

func reportErrors(acc *core.ErrorAccumulator) error {
	var msgs []string
	for _, e := range acc.Errors() {
		if e.Severity >= core.LocalFatal {
			fmt.Fprintf(os.Stderr, "fuego: %s\n", e.Error())
			msgs = append(msgs, e.Error())
		}
	}
	if len(msgs) == 1 {
		return fmt.Errorf("build failed: %s", msgs[0])
	}
	return fmt.Errorf("build failed with %d errors: %s", len(msgs), msgs[0])
}
