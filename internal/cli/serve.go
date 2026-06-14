package cli

import (
	"context"
	"fmt"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
	"github.com/gofuego/fuego/internal/pipeline"
	"github.com/gofuego/fuego/internal/serve"
	"github.com/spf13/cobra"
)

func newServeCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	var baseURL string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the development server with live reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if cmd.Flags().Changed("base-url") {
				cfg.Site.BaseURL = baseURL
			}
			return serve.Run(cfg, func() error {
				return doBuild(cfg, parsers, hooks, packs)
			})
		},
	}
	cmd.Flags().StringVar(&baseURL, "base-url", "",
		"override the site base_url (deploy subpath, e.g. /owner/repo); empty for root")
	return cmd
}

func doBuild(cfg *config.Config, parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) error {
	ctx := context.Background()
	// The dev server rebuilds on every change, so incremental parsing keeps
	// rebuilds fast on large sites.
	return pipeline.Build(ctx, cfg, parsers, hooks, packs, pipeline.Options{Incremental: true})
}
