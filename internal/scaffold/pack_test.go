package scaffold_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/internal/scaffold"
)

// TestScaffoldWithPack verifies that --pack wiring is generated offline (no
// network, no go run/build): main.go imports the pack and calls
// eng.Use(symbol.Pack()), and CLAUDE.md records the pack.
func TestScaffoldWithPack(t *testing.T) {
	dir := t.TempDir()
	err := scaffold.WriteFiles(dir, scaffold.Data{
		Name:       "site",
		Module:     "site",
		PackImport: "github.com/gofuego/fuego-adr/adr",
		PackSymbol: "adr",
	})
	if err != nil {
		t.Fatalf("WriteFiles: %v", err)
	}

	main, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(main)
	if !strings.Contains(got, `"github.com/gofuego/fuego-adr/adr"`) {
		t.Errorf("main.go missing pack import:\n%s", got)
	}
	if !strings.Contains(got, "eng.Use(adr.Pack())") {
		t.Errorf("main.go missing eng.Use wiring:\n%s", got)
	}

	claude, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(claude), "github.com/gofuego/fuego-adr/adr") {
		t.Error("CLAUDE.md should mention the installed pack")
	}
	if !strings.Contains(string(claude), "go run . config") {
		t.Error("CLAUDE.md pack note should point at `fuego config`")
	}
}

// TestScaffoldWithoutPackUnchanged confirms the default (no --pack) scaffold
// has no pack import or Use wiring.
func TestScaffoldWithoutPackUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "site", Module: "site"}); err != nil {
		t.Fatal(err)
	}
	main, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	// The conditional pack block (and its marker comment) must be absent.
	if strings.Contains(string(main), "Format pack installed via") {
		t.Errorf("plain scaffold should not include the pack block:\n%s", main)
	}
	if _, err := os.Stat(filepath.Join(dir, "CLAUDE.md")); err != nil {
		t.Fatalf("CLAUDE.md missing: %v", err)
	}
}
