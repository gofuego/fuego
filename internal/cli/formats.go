package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gofuego/fuego/internal/formats"
	"github.com/spf13/cobra"
)

// newFormatsCmd manages a project's format modules: `formats add` installs a
// format (dependency + registration in the tool-owned formats.go + contract
// docs under docs/formats/), `formats sync` refreshes the materialized docs
// from the versions go.mod pins.
func newFormatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "formats",
		Short: "Manage the project's format modules",
	}
	cmd.AddCommand(newFormatsAddCmd(), newFormatsSyncCmd())
	return cmd
}

func newFormatsAddCmd() *cobra.Command {
	var symbol string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Install a format module: dependency, registration, and contract docs",
		Long: `Install a format module into the current project.

Short names resolve by convention to github.com/gofuego/fuego-formats/<name>;
"markdown" resolves to the engine's first-party parser; a full package path
installs a third-party format. The format is added to the project's
dependencies, registered in the generated formats.go, and its contract docs
(schema.md + golden fixtures) are materialized under docs/formats/.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormatsAdd(cmd, args[0], symbol)
		},
	}
	cmd.Flags().StringVar(&symbol, "symbol", "",
		"package identifier for the format's Parser() call (default: the path's last segment)")
	return cmd
}

func newFormatsSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Refresh docs/formats/ from the pinned format module versions",
		Long: `Re-copy every installed format's contract docs (schema.md + golden
fixtures) from the module versions the project's go.mod pins. Run it after
upgrading a format module so the local contracts match the code.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormatsSync(cmd)
		},
	}
}

func runFormatsAdd(cmd *cobra.Command, name, symbol string) error {
	if err := requireProject(); err != nil {
		return err
	}
	f, err := formats.ResolveWithSymbol(name, symbol)
	if err != nil {
		return err
	}

	installed, parseErr := formats.ParseFile(".")
	if parseErr != nil && !os.IsNotExist(parseErr) {
		return fmt.Errorf("reading %s: %w (regenerate it by re-running with the file removed)", formats.FileName, parseErr)
	}
	for _, ex := range installed {
		if ex.Name == f.Name || ex.ImportPath == f.ImportPath {
			return fmt.Errorf("format %s (%s) is already installed", ex.Name, ex.ImportPath)
		}
	}

	if err := ensureDependency(f); err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if os.IsNotExist(parseErr) {
		// A project without the generated formats.go (hand-written main, or
		// scaffolded before the convention): materialize the docs and print
		// the wiring — never rewrite user code.
		if err := formats.MaterializeDocs(".", f); err != nil {
			return err
		}
		fmt.Fprintf(out, "fuego: installed %s and materialized its contract into %s/%s\n", f.Name, formats.DocsDir, f.Name)
		fmt.Fprintf(out, "\nThis project has no generated %s, so register the parser yourself:\n\n", formats.FileName)
		fmt.Fprintf(out, "  import %s %q\n\n", f.Symbol, f.ImportPath)
		fmt.Fprintf(out, "  eng.Register(%s.Parser())\n", f.Symbol)
		return nil
	}

	installed = append(installed, f)
	if err := formats.WriteFile(".", installed); err != nil {
		return err
	}
	if err := formats.MaterializeDocs(".", f); err != nil {
		return err
	}
	if err := formats.WriteIndex(".", installed); err != nil {
		return err
	}
	tidy()

	fmt.Fprintf(out, "fuego: installed format %s (%s)\n", f.Name, f.ImportPath)
	fmt.Fprintf(out, "  registered in %s; contract in %s/%s\n", formats.FileName, formats.DocsDir, f.Name)
	return nil
}

func runFormatsSync(cmd *cobra.Command) error {
	if err := requireProject(); err != nil {
		return err
	}
	installed, err := formats.ParseFile(".")
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no %s in this project — `fuego formats sync` refreshes the docs of formats registered there (install one with `fuego formats add <name>`)", formats.FileName)
		}
		return fmt.Errorf("reading %s: %w", formats.FileName, err)
	}

	out := cmd.OutOrStdout()
	var failed int
	for _, f := range installed {
		if err := formats.MaterializeDocs(".", f); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "fuego: warning: %s: %v\n", f.Name, err)
			continue
		}
		fmt.Fprintf(out, "fuego: refreshed %s/%s\n", formats.DocsDir, f.Name)
	}
	if err := formats.WriteIndex(".", installed); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d format(s) failed to sync", failed)
	}
	return nil
}

// requireProject checks the command runs at a Go project root.
func requireProject() error {
	if _, err := os.Stat("go.mod"); err != nil {
		return fmt.Errorf("no go.mod here — run from the project root")
	}
	return nil
}

// ensureDependency makes the format's package resolvable, fetching it only
// when needed (a locally-replaced or already-required module needs no
// network).
func ensureDependency(f formats.Format) error {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go toolchain not found in PATH: %w", err)
	}
	probe := exec.Command(goPath, "list", "-f", "{{.Dir}}", f.ImportPath)
	if err := probe.Run(); err == nil {
		return nil
	}
	get := exec.Command(goPath, "get", f.ImportPath+"@latest")
	get.Stdout = os.Stdout
	get.Stderr = os.Stderr
	if err := get.Run(); err != nil {
		return fmt.Errorf("go get %s: %w", f.ImportPath, err)
	}
	return nil
}

// tidy is best-effort go mod tidy after the registration file changes.
func tidy() {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return
	}
	cmd := exec.Command(goPath, "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
