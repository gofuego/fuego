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
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the development server with live reload",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			return serve.Run(cfg, func() error {
				return doBuild(cfg, parsers, hooks, packs)
			})
		},
	}
}

func doBuild(cfg *config.Config, parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) error {
	ctx := context.Background()
	// The dev server rebuilds on every change, so incremental parsing keeps
	// rebuilds fast on large sites.
	return pipeline.Build(ctx, cfg, parsers, hooks, packs, pipeline.Options{Incremental: true})
}
