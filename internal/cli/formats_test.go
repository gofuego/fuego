package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gofuego/fuego/internal/formats"
	"github.com/gofuego/fuego/internal/scaffold"
)

// fakeModule writes a minimal local format module (go.mod + package +
// schema.md + fixtures) and returns its dir. Directory replaces resolve it
// without any network — the tests run with GOPROXY=off to prove that.
func fakeModule(t *testing.T, modPath, pkgName string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module "+modPath+"\n\ngo 1.25\n")
	write(pkgName+".go", "package "+pkgName+"\n")
	write("schema.md", "# "+pkgName+" — parser contract\n")
	write("testdata/sample."+pkgName, "input\n")
	write("testdata/sample.golden.json", "{}\n")
	return dir
}

// formatsProject scaffolds a project whose go.mod requires only the given
// fake modules via directory replaces, then chdirs into it.
func formatsProject(t *testing.T, mods map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	md, err := formats.Resolve("markdown")
	if err != nil {
		t.Fatal(err)
	}
	if err := scaffold.WriteFiles(dir, scaffold.Data{Name: "demo", Module: "demo", Formats: []formats.Format{md}}); err != nil {
		t.Fatal(err)
	}

	gomod := "module demo\n\ngo 1.25\n"
	for path := range mods {
		gomod += "\nrequire " + path + " v0.0.0"
	}
	gomod += "\n"
	for path, local := range mods {
		gomod += "\nreplace " + path + " => " + local
	}
	gomod += "\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(dir)
	t.Setenv("GOPROXY", "off") // any network dependency is a test failure
	return dir
}

func snapshot(t *testing.T, dir string, files ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			t.Fatal(err)
		}
		out[f] = string(b)
	}
	return out
}

func TestFormatsAddRoundTrip(t *testing.T) {
	mod := fakeModule(t, "example.com/fakefmt", "fakefmt")
	dir := formatsProject(t, map[string]string{"example.com/fakefmt": mod})

	// Files `add` must not touch (the acceptance criterion).
	before := snapshot(t, dir, "main.go", "config.yaml", "CLAUDE.md", "theme/base.html")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runFormatsAdd(cmd, "example.com/fakefmt", ""); err != nil {
		t.Fatalf("add: %v", err)
	}

	// formats.go regenerated: markdown kept, fakefmt appended.
	ff, _ := os.ReadFile(filepath.Join(dir, formats.FileName))
	for _, want := range []string{
		`"github.com/gofuego/fuego/parsers/markdown"`,
		`"example.com/fakefmt"`,
		"eng.Register(fakefmt.Parser())",
	} {
		if !strings.Contains(string(ff), want) {
			t.Errorf("formats.go missing %q:\n%s", want, ff)
		}
	}

	// Docs materialized (from the read-only-ish module dir), index updated.
	for _, rel := range []string{
		"docs/formats/fakefmt/schema.md",
		"docs/formats/fakefmt/testdata/sample.golden.json",
		"docs/formats/README.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing %s: %v", rel, err)
		}
	}
	idx, _ := os.ReadFile(filepath.Join(dir, "docs", "formats", "README.md"))
	if !strings.Contains(string(idx), "fakefmt") || !strings.Contains(string(idx), "markdown") {
		t.Errorf("index must list both formats:\n%s", idx)
	}

	// Unrelated files untouched.
	after := snapshot(t, dir, "main.go", "config.yaml", "CLAUDE.md", "theme/base.html")
	for f, b := range before {
		if after[f] != b {
			t.Errorf("add touched unrelated file %s", f)
		}
	}

	// Adding the same format again is an error.
	if err := runFormatsAdd(cmd, "example.com/fakefmt", ""); err == nil || !strings.Contains(err.Error(), "already installed") {
		t.Errorf("duplicate add: want already-installed error, got %v", err)
	}
}

func TestFormatsSyncRefreshesDocs(t *testing.T) {
	mod := fakeModule(t, "example.com/fakefmt", "fakefmt")
	dir := formatsProject(t, map[string]string{"example.com/fakefmt": mod})

	// A project whose only installed format is the fake one (markdown would
	// need the engine module resolvable, which this offline project omits).
	f, err := formats.Resolve("example.com/fakefmt")
	if err != nil {
		t.Fatal(err)
	}
	if err := formats.WriteFile(dir, []formats.Format{f}); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runFormatsSync(cmd); err != nil {
		t.Fatalf("first sync: %v", err)
	}
	doc := filepath.Join(dir, "docs", "formats", "fakefmt", "schema.md")
	first, err := os.ReadFile(doc)
	if err != nil {
		t.Fatalf("sync did not materialize docs: %v", err)
	}

	// The module "upgrades" (its docs change); sync re-copies the pinned
	// version's docs.
	updated := "# fakefmt — parser contract (v2)\n"
	if err := os.WriteFile(filepath.Join(mod, "schema.md"), []byte(updated), 0644); err != nil {
		t.Fatal(err)
	}
	if err := runFormatsSync(cmd); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	second, _ := os.ReadFile(doc)
	if string(second) != updated {
		t.Errorf("sync did not refresh schema.md: got %q, had %q", second, first)
	}
	if _, err := os.Stat(filepath.Join(dir, "docs", "formats", "README.md")); err != nil {
		t.Errorf("sync must regenerate the index: %v", err)
	}
}

func TestFormatsAddWithoutFormatsFilePrintsWiring(t *testing.T) {
	mod := fakeModule(t, "example.com/fakefmt", "fakefmt")
	dir := formatsProject(t, map[string]string{"example.com/fakefmt": mod})
	if err := os.Remove(filepath.Join(dir, formats.FileName)); err != nil {
		t.Fatal(err)
	}
	mainBefore := snapshot(t, dir, "main.go")

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	if err := runFormatsAdd(cmd, "example.com/fakefmt", ""); err != nil {
		t.Fatalf("degraded add: %v", err)
	}

	// Docs still materialize; the wiring is printed, never written.
	if _, err := os.Stat(filepath.Join(dir, "docs", "formats", "fakefmt", "schema.md")); err != nil {
		t.Errorf("degraded add must still materialize docs: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "eng.Register(fakefmt.Parser())") {
		t.Errorf("degraded add must print the wiring:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, formats.FileName)); !os.IsNotExist(err) {
		t.Error("degraded add must not create formats.go")
	}
	if got := snapshot(t, dir, "main.go"); got["main.go"] != mainBefore["main.go"] {
		t.Error("degraded add touched main.go")
	}
}
