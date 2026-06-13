package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderResolvedProvenance(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(userPath, []byte(`site:
  name: My Site
routes:
  adr: /decisions/{slug}
`), 0644); err != nil {
		t.Fatal(err)
	}

	packLayer, err := ParsePackLayer("adr", []byte(`routes:
  adr: /adr/{slug}
taxonomies:
  status:
    path: /status/{term}
`))
	if err != nil {
		t.Fatal(err)
	}

	out, err := RenderResolved(userPath, []Layer{packLayer})
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	// User override wins and is attributed to the user.
	if !strings.Contains(got, "/decisions/{slug}") {
		t.Errorf("user route override missing:\n%s", got)
	}
	// Pack-only taxonomy is present and attributed to the pack.
	if !strings.Contains(got, "pack: adr") {
		t.Errorf("expected pack provenance comment:\n%s", got)
	}
	if !strings.Contains(got, "# user") {
		t.Errorf("expected user provenance comment:\n%s", got)
	}

	// Deterministic: keys sorted, repeated render identical.
	out2, _ := RenderResolved(userPath, []Layer{packLayer})
	if string(out2) != got {
		t.Error("RenderResolved is not deterministic")
	}
}

func TestRenderResolvedNoPacks(t *testing.T) {
	dir := t.TempDir()
	userPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(userPath, []byte("site:\n  name: Solo\n"), 0644)

	out, err := RenderResolved(userPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "# user") {
		t.Errorf("expected user provenance:\n%s", out)
	}
}
