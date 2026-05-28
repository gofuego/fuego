package cli

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/FabioSol/fuego/internal/scaffold"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init <name>",
		Short: "Scaffold a new Fuego project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// Check Go toolchain availability
			if _, err := exec.LookPath("go"); err != nil {
				return fmt.Errorf("go toolchain not found in PATH: %w", err)
			}

			dir, err := filepath.Abs(name)
			if err != nil {
				return fmt.Errorf("resolving path: %w", err)
			}

			data := scaffold.Data{
				Name:   filepath.Base(dir),
				Module: filepath.Base(dir),
			}

			if err := scaffold.Generate(dir, data); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "fuego: project %q created at %s\n", name, dir)
			fmt.Fprintf(cmd.OutOrStdout(), "  cd %s && go run . build\n", name)
			return nil
		},
	}
}
