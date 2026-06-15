package linkcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func TestResolve(t *testing.T) {
	const base = "/owner/repo/" // a typical <base href> under a deploy subpath
	cases := []struct {
		href   string
		want   string
		wantOK bool
	}{
		{"decisions/a/", "/owner/repo/decisions/a", true},     // base-relative
		{"/owner/repo/decisions/a/", "/owner/repo/decisions/a", true}, // absolute, prefixed
		{".", "/owner/repo", true},                            // self
		{"../b/", "/owner/b", true},                           // dot-dot
		{"/decisions/a/", "/decisions/a", true},               // absolute path escapes base
		{"https://x.com/a", "", false},                        // external
		{"//cdn.example/x", "", false},                        // protocol-relative
		{"mailto:a@b.c", "", false},                           // scheme
		{"#section", "", false},                               // in-page anchor
		{"", "", false},                                       // empty
		{"page/#frag", "/owner/repo/page", true},              // fragment stripped
	}
	for _, c := range cases {
		got, ok := resolve(base, c.href)
		if ok != c.wantOK || (ok && got != c.want) {
			t.Errorf("resolve(%q) = (%q, %v), want (%q, %v)", c.href, got, ok, c.want, c.wantOK)
		}
	}
}

func TestCheck(t *testing.T) {
	out := t.TempDir()
	const baseHref = `<base href="/owner/repo/">`

	write := func(rel, body string) {
		p := filepath.Join(out, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Home page links: two good, two broken, plus external/anchor (ignored).
	write("index.html", baseHref+`
		<a href="decisions/a/">good base-relative</a>
		<a href="/owner/repo/decisions/a/">good absolute</a>
		<a href="/decisions/a/">broken: escapes base</a>
		<a href="missing/">broken: dangling</a>
		<a href="https://example.com">external</a>
		<a href="#top">anchor</a>`)
	// "." resolves against <base href> to the home page (valid). A "../../"
	// here would escape the base and be (correctly) flagged — see TestResolve.
	write("decisions/a/index.html", baseHref+`<a href=".">home</a>`)
	write("style.css", "body{}")

	pages := []*core.Page{
		{URL: "/", RelPath: "index.md"},
		{URL: "/decisions/a/", RelPath: "decisions/a.md"},
	}
	cfg := &config.Config{}
	cfg.Site.BaseURL = "/owner/repo"
	cfg.Dirs.Output = out

	errs := Check(pages, cfg, core.Warning)
	if len(errs) != 2 {
		t.Fatalf("got %d broken links, want 2:\n%s", len(errs), errsString(errs))
	}
	for _, e := range errs {
		if e.Severity != core.Warning {
			t.Errorf("severity = %v, want Warning", e.Severity)
		}
		if e.File != "index.md" {
			t.Errorf("broken link attributed to %q, want index.md", e.File)
		}
	}
	joined := errsString(errs)
	if !strings.Contains(joined, "/decisions/a") || !strings.Contains(joined, "/owner/repo/missing") {
		t.Errorf("error messages missing expected targets:\n%s", joined)
	}
}

func TestCheckCleanSite(t *testing.T) {
	out := t.TempDir()
	os.WriteFile(filepath.Join(out, "index.html"),
		[]byte(`<base href="/"><a href="about/">about</a><a href="/about/">about2</a>`), 0o644)
	os.MkdirAll(filepath.Join(out, "about"), 0o755)
	os.WriteFile(filepath.Join(out, "about", "index.html"), []byte(`<a href=".">x</a>`), 0o644)

	pages := []*core.Page{{URL: "/", RelPath: "index.md"}, {URL: "/about/", RelPath: "about.md"}}
	cfg := &config.Config{}
	cfg.Dirs.Output = out // base_url empty (root deploy)

	if errs := Check(pages, cfg, core.GlobalFatal); len(errs) != 0 {
		t.Errorf("clean site reported broken links: %s", errsString(errs))
	}
}

func errsString(errs []core.EngineError) string {
	var b strings.Builder
	for i := range errs {
		b.WriteString("  " + errs[i].Error() + "\n")
	}
	return b.String()
}
