package cli

import (
	"github.com/FabioSol/fuego/core"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

// Execute runs the Cobra command tree with the given arguments and parser registry.
func Execute(args []string, parsers map[string]core.Parser) error {
	root := newRootCmd(parsers)
	root.SetArgs(args[1:]) // strip the binary name
	return root.Execute()
}

func newRootCmd(parsers map[string]core.Parser) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:     "fuego",
		Short:   "A meta-engine for static site generation",
		Version: Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to configuration file")

	cmd.AddCommand(
		newBuildCmd(parsers, &configPath),
		newValidateCmd(parsers, &configPath),
		newListCmd(parsers, &configPath),
		newServeCmd(parsers, &configPath),
		newInitCmd(),
	)

	return cmd
}
