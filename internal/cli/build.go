package cli

import (
	"context"
	"fmt"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/pipeline"
	"github.com/spf13/cobra"
)

func newBuildCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack, configPath *string) *cobra.Command {
	var incremental bool
	var baseURL string
	var checkLinks bool
	var strictLinks bool
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the static site",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(*configPath, packs)
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}
			if cmd.Flags().Changed("base-url") {
				cfg.Site.BaseURL = baseURL
			}

			ctx := context.Background()
			return pipeline.Build(ctx, cfg, parsers, hooks, packs, pipeline.Options{
				Incremental: incremental,
				CheckLinks:  checkLinks || strictLinks,
				StrictLinks: strictLinks,
			})
		},
	}
	cmd.Flags().BoolVar(&incremental, "incremental", false,
		"reuse cached parses for unchanged content (falls back to a full rebuild if the binary, config, or theme changed)")
	cmd.Flags().StringVar(&baseURL, "base-url", "",
		"override the site base_url (deploy subpath, e.g. /owner/repo); empty for root")
	cmd.Flags().BoolVar(&checkLinks, "check-links", false,
		"after building, report internal links that don't resolve to a generated page (run with --base-url to catch base-path mistakes)")
	cmd.Flags().BoolVar(&strictLinks, "strict-links", false,
		"fail the build on a broken internal link (implies --check-links)")
	return cmd
}
