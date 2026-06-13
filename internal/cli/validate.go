package cli

import (
	"context"
	"fmt"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/spf13/cobra"
)

func newValidateCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate content without rendering (for CI)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			ctx := context.Background()
			res, err := pipeline.RunUntil(ctx, cfg, parsers, hooks, packs, pipeline.PhaseIndex)
			if err != nil {
				return err
			}

			if res.Errors.HasFatal() {
				for _, e := range res.Errors.Errors() {
					if e.Severity >= core.LocalFatal {
						fmt.Fprintf(cmd.ErrOrStderr(), "fuego: %s\n", e.Error())
					}
				}
				return fmt.Errorf("validation failed")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "fuego: %d pages validated successfully\n", len(res.Pages))
			return nil
		},
	}
}
