package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/spf13/cobra"
)

func newListCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all pages with their types and URLs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*configPath)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			ctx := context.Background()
			res, err := pipeline.RunUntil(ctx, cfg, parsers, hooks, packs, pipeline.PhaseIndex)
			if err != nil {
				return err
			}

			printPageTable(cmd.OutOrStdout(), res)
			return nil
		},
	}
}

func printPageTable(w io.Writer, res *pipeline.Result) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TYPE\tSOURCE\tURL")
	for _, p := range res.Pages {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", p.Type, p.RelPath, p.URL)
	}
	tw.Flush()
}
