package cli

import (
	"context"
	"fmt"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/spf13/cobra"
)

func newBuildCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			ctx := context.Background()
			return pipeline.Build(ctx, cfg, parsers, hooks, packs)
		},
	}
}
