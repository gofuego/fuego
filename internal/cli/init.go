package cli

import (
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gofuego/fuego/internal/formats"
	"github.com/gofuego/fuego/internal/scaffold"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var packModule string
	var packSymbol string
	var formatList string

	cmd := &cobra.Command{
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

			fmts, err := resolveFormatList(formatList)
			if err != nil {
				return err
			}

			data := scaffold.Data{
				Name:    filepath.Base(dir),
				Module:  filepath.Base(dir),
				Formats: fmts,
			}

			if packModule != "" {
				symbol := packSymbol
				if symbol == "" {
					symbol = path.Base(packModule)
				}
				if !isGoIdentifier(symbol) {
					return fmt.Errorf("cannot derive a package name from %q (got %q); pass --pack-symbol with the pack's package name",
						packModule, symbol)
				}
				data.PackImport = packModule
				data.PackSymbol = symbol
			}

			if err := scaffold.Generate(dir, data); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "fuego: project %q created at %s\n", name, dir)
			for _, f := range fmts {
				fmt.Fprintf(cmd.OutOrStdout(), "  installed format %s (%s; contract in %s/%s)\n",
					f.Name, f.ImportPath, formats.DocsDir, f.Name)
			}
			if packModule != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "  installed pack %s (wired in main.go)\n", packModule)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "  cd %s && go run . build\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&formatList, "formats", "markdown",
		"comma-separated format modules to register — short names (openapi, dbml, …), the markdown alias, or full package paths; the list is the exact set")
	cmd.Flags().StringVar(&packModule, "pack", "",
		"format pack module to install and wire in (e.g. github.com/gofuego/fuego-adr/adr)")
	cmd.Flags().StringVar(&packSymbol, "pack-symbol", "",
		"package identifier for the pack's Pack() call (default: the module's last path segment)")

	return cmd
}

// resolveFormatList resolves a comma-separated --formats value into the exact
// format set, rejecting duplicates by name.
func resolveFormatList(list string) ([]formats.Format, error) {
	var out []formats.Format
	seen := map[string]bool{}
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		f, err := formats.Resolve(name)
		if err != nil {
			return nil, fmt.Errorf("--formats %s: %w (third-party paths needing a custom identifier: init first, then `fuego formats add %s --symbol <name>`)", name, err, name)
		}
		if seen[f.Name] {
			return nil, fmt.Errorf("--formats lists %s twice", f.Name)
		}
		seen[f.Name] = true
		out = append(out, f)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("--formats resolved to an empty set")
	}
	return out, nil
}

// isGoIdentifier reports whether s is a valid Go identifier, so it can be used
// as the package selector in the generated main.go.
func isGoIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		switch {
		case r == '_' || unicode.IsLetter(r):
		case i > 0 && unicode.IsDigit(r):
		default:
			return false
		}
	}
	return !isGoKeyword(s)
}

func isGoKeyword(s string) bool {
	switch s {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else",
		"fallthrough", "for", "func", "go", "goto", "if", "import", "interface",
		"map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return true
	}
	return false
}
