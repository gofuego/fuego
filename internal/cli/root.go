package cli

import (
	"runtime/debug"

	"github.com/gofuego/fuego/core"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags. When unset, resolveVersion falls
// back to the module version embedded by `go install module@version`.
var Version = "dev"

// resolveVersion returns the ldflags-injected version if present, otherwise the
// module version Go records in the build info (so `go install …@v0.4.1` reports
// v0.4.1, not "dev"). A plain `go build` checkout reports "dev".
func resolveVersion() string {
	if Version != "dev" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return Version
}

// Execute runs the Cobra command tree with the given arguments, parser
// registry, hooks, and registered packs.
func Execute(args []string, parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) error {
	root := newRootCmd(parsers, hooks, packs)
	root.SetArgs(args[1:]) // strip the binary name
	return root.Execute()
}

func newRootCmd(parsers map[string]core.Parser, hooks *core.Hooks, packs []core.Pack) *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:     "fuego",
		Short:   "A meta-engine for static site generation",
		Version: resolveVersion(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to configuration file")

	cmd.AddCommand(
		newBuildCmd(parsers, hooks, packs, &configPath),
		newValidateCmd(parsers, hooks, packs, &configPath),
		newListCmd(parsers, hooks, packs, &configPath),
		newServeCmd(parsers, hooks, packs, &configPath),
		newConfigCmd(packs, &configPath),
		newInitCmd(),
	)

	return cmd
}
