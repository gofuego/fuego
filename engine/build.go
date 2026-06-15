package engine

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/gofuego/fuego/internal/config"
	"github.com/gofuego/fuego/internal/pipeline"
	"github.com/gofuego/fuego/internal/serve"
)

// BuildOptions configures a build, serve, or validate driven through the Go
// API instead of a config.yaml file — the path domain-specific tools (and
// pack-based wrappers) use to build in-process without writing a temp config.
//
// Resolution order, lowest precedence first: registered packs' config defaults
// → ConfigPath file (if set) → the non-zero fields below. So routes,
// taxonomies, and collections typically come from a pack, while a wrapper sets
// just the dirs and site here.
type BuildOptions struct {
	ConfigPath string // optional base config.yaml to start from

	ContentDir string // content directory (default "content")
	ThemeDir   string // user theme directory; packs may supply the theme instead
	OutputDir  string // build output directory (default "build")
	StaticDir  string // public/static directory (default "public")

	SiteName string
	BaseURL  string

	DevPort    int    // dev server port (Serve)
	DevCommand string // dev subprocess, e.g. a Vite command (Serve)
	ProxyPort  int    // dev asset proxy port (Serve)

	Incremental bool   // reuse cached parses for unchanged content
	CacheDir    string // build cache directory (default ".fuego")

	CheckLinks  bool // after building, report internal links that don't resolve
	StrictLinks bool // fail the build on a broken internal link (implies CheckLinks)
}

// Build runs the full build pipeline with the resolved configuration.
func (e *Engine) Build(ctx context.Context, opts BuildOptions) error {
	cfg, err := e.resolveConfig(opts)
	if err != nil {
		return err
	}
	return pipeline.Build(ctx, cfg, e.parsers, &e.hooks, e.packs, pipeline.Options{
		Incremental: opts.Incremental,
		CacheDir:    opts.CacheDir,
		CheckLinks:  opts.CheckLinks || opts.StrictLinks,
		StrictLinks: opts.StrictLinks,
	})
}

// Validate runs the pipeline through INDEX without rendering and returns the
// number of pages that would be produced. A fatal error fails validation.
func (e *Engine) Validate(ctx context.Context, opts BuildOptions) (int, error) {
	cfg, err := e.resolveConfig(opts)
	if err != nil {
		return 0, err
	}
	res, err := pipeline.RunUntil(ctx, cfg, e.parsers, &e.hooks, e.packs, nil, pipeline.PhaseIndex)
	if err != nil {
		return 0, err
	}
	if res.Errors.HasFatal() {
		return 0, fmt.Errorf("validation failed")
	}
	return len(res.Pages), nil
}

// Serve starts the development server — an initial build, a file watcher that
// rebuilds on change, and an HTTP server. It blocks until interrupted.
func (e *Engine) Serve(ctx context.Context, opts BuildOptions) error {
	cfg, err := e.resolveConfig(opts)
	if err != nil {
		return err
	}
	buildOpts := pipeline.Options{Incremental: true, CacheDir: opts.CacheDir}
	return serve.Run(cfg, func() error {
		return pipeline.Build(ctx, cfg, e.parsers, &e.hooks, e.packs, buildOpts)
	})
}

// resolveConfig builds the layered config: pack defaults, then an optional
// file, then the option overrides on top.
func (e *Engine) resolveConfig(opts BuildOptions) (*config.Config, error) {
	var layers []config.Layer
	for _, p := range e.packs {
		if len(p.ConfigDefaults) == 0 {
			continue
		}
		l, err := config.ParsePackLayer(p.Name, p.ConfigDefaults)
		if err != nil {
			return nil, err
		}
		layers = append(layers, l)
	}

	if opts.ConfigPath != "" {
		data, err := os.ReadFile(opts.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("reading config: %w", err)
		}
		m := map[string]any{}
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parsing config: %w", err)
		}
		layers = append(layers, config.Layer{Source: "file", Data: m})
	}

	layers = append(layers, config.Layer{Source: "options", Data: optionOverrides(opts)})

	cfg, _, err := config.ResolveLayers(layers)
	return cfg, err
}

func optionOverrides(opts BuildOptions) map[string]any {
	m := map[string]any{}

	site := map[string]any{}
	if opts.SiteName != "" {
		site["name"] = opts.SiteName
	}
	if opts.BaseURL != "" {
		site["base_url"] = opts.BaseURL
	}
	if len(site) > 0 {
		m["site"] = site
	}

	dirs := map[string]any{}
	for k, v := range map[string]string{
		"content": opts.ContentDir,
		"theme":   opts.ThemeDir,
		"output":  opts.OutputDir,
		"static":  opts.StaticDir,
	} {
		if v != "" {
			dirs[k] = v
		}
	}
	if len(dirs) > 0 {
		m["dirs"] = dirs
	}

	dev := map[string]any{}
	if opts.DevPort != 0 {
		dev["port"] = opts.DevPort
	}
	if opts.DevCommand != "" {
		dev["command"] = opts.DevCommand
	}
	if opts.ProxyPort != 0 {
		dev["proxy_port"] = opts.ProxyPort
	}
	if len(dev) > 0 {
		m["dev"] = dev
	}

	return m
}
