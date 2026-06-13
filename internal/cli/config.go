package cli

import (
	"fmt"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCmd(packs []core.Pack, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Print the resolved configuration with per-key provenance",
		Long: "Print the fully merged configuration — your config.yaml deep-merged " +
			"with every registered pack's defaults — annotated with where each value " +
			"came from (# user or # pack: name).",
		RunE: func(cmd *cobra.Command, args []string) error {
			layers, err := packLayers(packs)
			if err != nil {
				return err
			}
			out, err := config.RenderResolved(*configPath, layers)
			if err != nil {
				return fmt.Errorf("resolving config: %w", err)
			}
			cmd.OutOrStdout().Write(out)
			return nil
		},
	}
}
