package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInitFormatsEndToEnd is the acceptance run for `fuego init --formats`:
// it scaffolds demo with markdown,mermaid,openapi, then proves `go build`
// and the project's own build command succeed with all three formats
// registered, and that the contracts materialized into docs/formats/.
//
// It fetches real modules, so it is gated: set FUEGO_NETWORK_TESTS=1.
func TestInitFormatsEndToEnd(t *testing.T) {
	if os.Getenv("FUEGO_NETWORK_TESTS") == "" {
		t.Skip("network test; set FUEGO_NETWORK_TESTS=1 to run")
	}

	repoRoot := fuegoRepoRoot(t)
	tmp := t.TempDir()
	t.Chdir(tmp)

	cmd := newInitCmd()
	cmd.SetArgs([]string{"demo", "--formats", "markdown,mermaid,openapi"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	dir := filepath.Join(tmp, "demo")

	// Exercise THIS engine, not the last release.
	runGo(t, dir, "mod", "edit", "-replace", "github.com/gofuego/fuego="+repoRoot)

	// TEMPORARY until formatkit/v0.2.0 is tagged: openapi's go.mod requires
	// that tag, so resolve formatkit to its develop pseudo-version. Delete
	// this block once the tag exists.
	if pseudo := goListVersion(t, dir, "github.com/gofuego/fuego-formats/formatkit@develop"); pseudo != "" {
		runGo(t, dir, "mod", "edit",
			"-replace", "github.com/gofuego/fuego-formats/formatkit=github.com/gofuego/fuego-formats/formatkit@"+pseudo)
	}
	runGo(t, dir, "mod", "tidy")

	// Docs may have failed to materialize during init (pre-tag resolution);
	// sync must complete them now that dependencies resolve.
	t.Chdir(dir)
	syncCmd := newFormatsSyncCmd()
	if err := syncCmd.RunE(syncCmd, nil); err != nil {
		t.Fatalf("formats sync: %v", err)
	}
	for _, rel := range []string{
		"docs/formats/markdown/schema.md",
		"docs/formats/mermaid/schema.md",
		"docs/formats/openapi/schema.md",
		"docs/formats/openapi/testdata",
		"docs/formats/README.md",
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing materialized doc %s: %v", rel, err)
		}
	}

	// `go build` succeeds with all three formats registered.
	runGo(t, dir, "build", "./...")

	// The project's own build command renders content of all three formats.
	write := func(rel, body string) {
		full := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(full), 0755)
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("content/flow.mmd", "graph TD\n  A-->B\n")
	write("content/api.openapi.yaml", `openapi: 3.0.3
info: {title: Demo API, version: "1"}
paths:
  /ping:
    get:
      operationId: ping
      tags: [Ops]
      responses: {"200": {description: ok}}
`)

	out := exec.Command(filepath.Join(dir, "demo"), "build")
	out.Dir = dir
	if b, err := out.CombinedOutput(); err != nil {
		t.Fatalf("demo build: %v\n%s", err, b)
	}
	for _, rel := range []string{
		"build/index.html",                     // markdown (scaffold home page)
		"build/flow/index.html",                // mermaid
		"build/api.openapi/index.html",         // openapi tree root
		"build/api.openapi/tags/ops/index.html", // openapi tree child
	} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("missing build output %s: %v", rel, err)
		}
	}
}

func fuegoRepoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatal(err)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == os.DevNull {
		t.Fatal("cannot locate the fuego repo root (go env GOMOD)")
	}
	return filepath.Dir(gomod)
}

func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, b)
	}
}

// goListVersion resolves a module@query to a concrete version, or "" if the
// query fails (e.g. the branch is gone once tags exist).
func goListVersion(t *testing.T, dir, modQuery string) string {
	t.Helper()
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Version}}", modQuery)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
