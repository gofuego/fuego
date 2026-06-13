package pipeline

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/buildcache"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/discover"
	"github.com/FabioSol/fuego/internal/index"
	"github.com/FabioSol/fuego/internal/manifest"
	"github.com/FabioSol/fuego/internal/parse"
	"github.com/FabioSol/fuego/internal/render"
	"github.com/FabioSol/fuego/internal/route"
)

// Options controls a build invocation.
type Options struct {
	// Incremental reuses parsed pages from a prior build whose content is
	// unchanged, and updates the output in place (removing orphaned files)
	// instead of cleaning it. A change to the engine binary, resolved config,
	// or theme falls back to a full rebuild.
	Incremental bool
	// CacheDir holds the build cache. Defaults to ".fuego" when empty.
	CacheDir string
}

func (o Options) cachePath() string {
	dir := o.CacheDir
	if dir == "" {
		dir = ".fuego"
	}
	return filepath.Join(dir, "cache.bin")
}

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
	Pages       []*core.Page
	AssetFiles  []discover.FileEntry
	Errors      *core.ErrorAccumulator
	ParsedPages map[string]buildcache.ParsedPage // fresh parse cache covering current files
	CacheStats  parse.CacheStats
}

// Build executes the full build pipeline:
// INIT → DISCOVER → PARSE → ROUTE → INDEX → RENDER → STATIC
func Build(ctx context.Context, cfg *config.Config, compiledParsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, opts Options) error {
	// === PREBUILD ===
	if err := runPrebuild(cfg.Prebuild); err != nil {
		return err
	}

	// === CACHE ===
	// Decide whether parsed pages from a prior build may be reused. A change
	// to the engine binary, resolved config, or theme invalidates the cache
	// and falls back to a full, clean rebuild.
	var prevCache *buildcache.Cache
	var prevParsed map[string]buildcache.ParsedPage
	header := buildHeader(cfg, packs)
	incremental := opts.Incremental && header.BinaryID != ""
	if incremental {
		if c, ok := buildcache.Load(opts.cachePath()); ok && c.Valid(header) {
			prevCache = c
			prevParsed = c.Pages
		}
	}

	// Full rebuild cleans the output dir; an incremental rebuild with a usable
	// cache updates it in place and removes orphans afterward.
	if prevCache == nil {
		if err := render.CleanOutput(cfg.Dirs.Output); err != nil {
			return fmt.Errorf("cleaning output directory: %w", err)
		}
	}

	res, err := RunUntil(ctx, cfg, compiledParsers, hooks, packs, prevParsed, PhaseIndex)
	if err != nil {
		return err
	}

	if res.Errors.HasFatal() {
		return reportErrors(res.Errors)
	}

	// Pages marked Skip stay in the pipeline result (hooks and tooling can
	// see them) but are excluded from output.
	renderable := res.Pages
	if hasSkipped(res.Pages) {
		renderable = make([]*core.Page, 0, len(res.Pages))
		for _, p := range res.Pages {
			if !p.Skip {
				renderable = append(renderable, p)
			}
		}
	}

	// === RENDER ===
	// A warm incremental rebuild (prevCache != nil) narrows rendering to the
	// affected set; a full/cold build renders everything.
	renderErrs := render.RenderAll(ctx, renderable, cfg, packs, res.CacheStats.Changed, prevCache != nil)
	for _, e := range renderErrs {
		res.Errors.Add(e)
	}

	if res.Errors.HasFatal() {
		return reportErrors(res.Errors)
	}

	// === OUTPUTS ===
	// Site-level non-HTML outputs (feeds, sitemaps) from theme/outputs/.
	outputErrs := render.RenderOutputs(renderable, cfg, packs)
	for _, e := range outputErrs {
		res.Errors.Add(e)
	}

	if res.Errors.HasFatal() {
		return reportErrors(res.Errors)
	}

	// === MANIFEST ===
	m := manifest.Generate(renderable, cfg)
	if err := manifest.Write(m, cfg.Dirs.Output); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	// === ORPHAN REMOVAL ===
	// On an incremental rebuild the output dir was not cleaned, so pages that
	// no longer exist must have their output files removed.
	newOutputs := outputRelPaths(renderable)
	if prevCache != nil {
		orphans := buildcache.OrphanedOutputs(prevCache.Outputs, newOutputs)
		for _, rel := range orphans {
			os.Remove(filepath.Join(cfg.Dirs.Output, rel))
		}
		buildcache.PruneEmptyDirs(cfg.Dirs.Output, orphans)
	}

	// === STATIC ===
	// Pack static assets first, then the user's public/ dir (user wins on
	// conflict), then content-colocated assets.
	if err := render.CopyPackStatic(packs, cfg.Dirs.Output); err != nil {
		return fmt.Errorf("copying pack static files: %w", err)
	}
	if err := render.CopyPublicDir(cfg.Dirs.Static, cfg.Dirs.Output); err != nil {
		return fmt.Errorf("copying static files: %w", err)
	}

	if len(res.AssetFiles) > 0 {
		if err := render.CopyAssets(res.AssetFiles, cfg.Dirs.Content, cfg.Dirs.Output); err != nil {
			return fmt.Errorf("copying assets: %w", err)
		}
	}

	// === SAVE CACHE ===
	if opts.Incremental && header.BinaryID != "" {
		nc := buildcache.New(header)
		nc.Pages = res.ParsedPages
		nc.Outputs = newOutputs
		if err := buildcache.Save(opts.cachePath(), nc); err != nil {
			fmt.Fprintf(os.Stderr, "fuego: warning: could not write build cache: %v\n", err)
		}
	}

	// Report warnings
	for _, e := range res.Errors.Errors() {
		if e.Severity == core.Warning {
			fmt.Fprintf(os.Stderr, "fuego: %s\n", e.Error())
		}
	}

	if prevCache != nil {
		fmt.Printf("fuego: built %d pages (%d reparsed, %d cached)\n",
			len(renderable), res.CacheStats.Parsed, res.CacheStats.Reused)
	} else {
		fmt.Printf("fuego: built %d pages\n", len(renderable))
	}
	return nil
}

// buildHeader computes the cache header for the current build environment.
func buildHeader(cfg *config.Config, packs []core.Pack) buildcache.Header {
	binID, _ := buildcache.BinaryID()
	cfgBytes, _ := yaml.Marshal(cfg)
	var packThemes []fs.FS
	for _, p := range packs {
		packThemes = append(packThemes, p.Theme)
	}
	return buildcache.Header{
		BinaryID:   binID,
		ConfigHash: buildcache.HashBytes(cfgBytes),
		ThemeHash:  buildcache.ThemeHash(cfg.Dirs.Theme, packThemes),
	}
}

// outputRelPaths returns the output index.html paths for the renderable pages.
func outputRelPaths(pages []*core.Page) []string {
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, buildcache.OutputRelPath(p.URL))
	}
	return out
}

// runPackInit calls each pack's Init (if set), merging parsers and hooks it
// registers, and warns about packs.{name} config subtrees with no matching
// registered pack. An Init error halts the build, naming the pack.
func runPackInit(ctx context.Context, packs []core.Pack, cfg *config.Config, parsers map[string]core.Parser, hooks *core.Hooks, acc *core.ErrorAccumulator) error {
	known := make(map[string]bool, len(packs))
	for _, p := range packs {
		known[p.Name] = true
	}
	for name := range cfg.Packs {
		if !known[name] {
			acc.Add(core.EngineError{
				Phase:    "INIT",
				Severity: core.Warning,
				Err:      fmt.Errorf("config has packs.%s but no pack named %q is registered", name, name),
			})
		}
	}

	for _, p := range packs {
		if p.Init == nil {
			continue
		}
		pc := core.NewPackContext(p.Name, cfg.Packs[p.Name])
		if err := p.Init(ctx, pc); err != nil {
			return fmt.Errorf("pack %q: %w", p.Name, err)
		}
		newParsers, newHooks := pc.Registered()
		for _, parser := range newParsers {
			parsers[parser.Type()] = parser
		}
		if hooks != nil {
			hooks.AfterParse = append(hooks.AfterParse, newHooks.AfterParse...)
			hooks.Index = append(hooks.Index, newHooks.Index...)
			hooks.BeforeRender = append(hooks.BeforeRender, newHooks.BeforeRender...)
		}
	}
	return nil
}

func hasSkipped(pages []*core.Page) bool {
	for _, p := range pages {
		if p.Skip {
			return true
		}
	}
	return false
}

// RunUntil executes pipeline phases up to and including the given phase.
// It does NOT clean the output directory — callers that render must do that themselves.
func RunUntil(ctx context.Context, cfg *config.Config, compiledParsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, prevParsed map[string]buildcache.ParsedPage, until Phase) (*Result, error) {
	acc := &core.ErrorAccumulator{}

	// === INIT ===
	// Two-tier parser merge: declarative (lowest) → compiled (highest).
	// Pack-declared parsers are already in compiledParsers (via engine.Use).
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

	// Pack Init lifecycle: run before DISCOVER so Init-registered parsers
	// participate in content classification. Init reads the pack's config
	// subtree and may register additional parsers and hooks.
	if err := runPackInit(ctx, packs, cfg, parsers, hooks, acc); err != nil {
		return nil, err
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
	pages, parseErrs, parsedMap, stats := parse.ParseAllCached(ctx, contentFiles, parsers, prevParsed)
	res.Pages = pages
	res.ParsedPages = parsedMap
	res.CacheStats = stats
	for _, e := range parseErrs {
		acc.Add(e)
	}

	if acc.HasGlobalFatal() {
		return res, reportErrors(acc)
	}

	if len(parseErrs) > 0 {
		for _, e := range parseErrs {
			fmt.Fprintf(os.Stderr, "fuego: %s\n", e.Error())
		}
	}

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
	virtualAdded := len(taxPages)+len(colPages) > 0

	// Index hooks run before the collision re-check so any virtual pages
	// they add are validated like engine-generated ones.
	if hooks != nil && len(hooks.Index) > 0 {
		for _, hook := range hooks.Index {
			pages, err = hook(pages)
			if err != nil {
				res.Pages = pages
				return res, fmt.Errorf("index hook: %w", err)
			}
		}
		virtualAdded = true
	}
	res.Pages = pages

	if virtualAdded {
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
