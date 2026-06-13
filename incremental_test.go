package fuego_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/FabioSol/fuego/parsers/markdown"
)

// TestIncrementalEquivalence is the safety contract for `build --incremental`:
// after any mutation, an incremental rebuild must produce a byte-identical
// output tree to a clean build of the same inputs. It exercises the full
// mutation matrix — no-op, content edit, add, delete, theme touch, config
// touch — against a controlled multi-format site.
func TestIncrementalEquivalence(t *testing.T) {
	input := t.TempDir()
	writeControlledSite(t, input)

	out := t.TempDir()      // incremental output, persists across mutations
	cacheDir := t.TempDir() // build cache, persists across mutations

	// Cold incremental build establishes the cache and output.
	buildSite(t, input, out, cacheDir, true)
	assertCleanParity(t, input, out)

	mutations := []struct {
		name string
		mut  func(t *testing.T, input string)
	}{
		{"noop", func(t *testing.T, in string) {}},
		{"edit-content", func(t *testing.T, in string) {
			appendFile(t, filepath.Join(in, "content/posts/alpha.md"), "\n\nAn edit.\n")
		}},
		{"add-content", func(t *testing.T, in string) {
			write(t, filepath.Join(in, "content/posts/delta.md"),
				"---\ntitle: Delta\ntags: [go]\n---\n\nNew page.\n")
		}},
		{"delete-content", func(t *testing.T, in string) {
			os.Remove(filepath.Join(in, "content/posts/beta.md"))
		}},
		{"touch-theme", func(t *testing.T, in string) {
			appendFile(t, filepath.Join(in, "theme/base.html"), "<!-- touched -->")
		}},
		{"touch-config", func(t *testing.T, in string) {
			write(t, filepath.Join(in, "config.yaml"), controlledConfig("Renamed Site"))
		}},
	}

	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			m.mut(t, input)
			buildSite(t, input, out, cacheDir, true) // incremental, in place
			assertCleanParity(t, input, out)
		})
	}
}

// assertCleanParity builds the current input cleanly into a throwaway dir and
// asserts the incremental output tree at `out` is byte-identical.
func assertCleanParity(t *testing.T, input, out string) {
	t.Helper()
	ref := t.TempDir()
	buildSite(t, input, ref, t.TempDir(), false)

	want := snapshotTree(t, ref)
	got := snapshotTree(t, out)

	for rel, wb := range want {
		gb, ok := got[rel]
		if !ok {
			t.Errorf("incremental output missing %s", rel)
			continue
		}
		if gb != wb {
			t.Errorf("file %s differs between incremental and clean build", rel)
		}
	}
	for rel := range got {
		if _, ok := want[rel]; !ok {
			t.Errorf("incremental output has stale file %s (not in clean build)", rel)
		}
	}
}

// TestIncrementalFixtureParity sweeps every integration fixture through the
// cache-reuse path: a cold incremental build writes the cache, a warm build
// reuses it, and the warm output must be byte-identical to a clean build. This
// catches any serialization-fidelity loss across the full range of real
// fixture content (envelopes, node trees, taxonomies, collections, packs).
func TestIncrementalFixtureParity(t *testing.T) {
	fixtures, _ := filepath.Glob("testdata/*")
	for _, fixture := range fixtures {
		info, err := os.Stat(fixture)
		if err != nil || !info.IsDir() {
			continue
		}
		name := filepath.Base(fixture)
		inputDir := filepath.Join(fixture, "input")
		if _, err := os.Stat(filepath.Join(fixture, "expected_error")); err == nil {
			continue // error fixtures never produce output
		}
		if _, err := os.Stat(filepath.Join(inputDir, "config.yaml")); err != nil {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			packs := fixturePacks(name)
			parsers := fixtureParserRegistry(name)
			hooks := fixtureHooks(name)

			build := func(outputDir, cacheDir string, incremental bool) {
				cfg, _, err := config.LoadLayered(filepath.Join(inputDir, "config.yaml"), fixturePackLayers(packs))
				if err != nil {
					t.Fatalf("loading config: %v", err)
				}
				cfg.Dirs.Content = filepath.Join(inputDir, cfg.Dirs.Content)
				cfg.Dirs.Theme = filepath.Join(inputDir, cfg.Dirs.Theme)
				cfg.Dirs.Static = filepath.Join(inputDir, cfg.Dirs.Static)
				cfg.Dirs.Output = outputDir
				if err := pipeline.Build(context.Background(), cfg, parsers, hooks, packs, pipeline.Options{
					Incremental: incremental, CacheDir: cacheDir,
				}); err != nil {
					t.Fatalf("build failed: %v", err)
				}
			}

			ref := t.TempDir()
			build(ref, t.TempDir(), false)

			incr, cache := t.TempDir(), t.TempDir()
			build(incr, cache, true) // cold: writes cache
			build(incr, cache, true) // warm: reuses cache

			want, got := snapshotTree(t, ref), snapshotTree(t, incr)
			for rel, wb := range want {
				if got[rel] != wb {
					t.Errorf("%s: warm-incremental output differs from clean build", rel)
				}
			}
			for rel := range got {
				if _, ok := want[rel]; !ok {
					t.Errorf("%s: warm-incremental has a file absent from the clean build", rel)
				}
			}
		})
	}
}

func buildSite(t *testing.T, inputDir, outputDir, cacheDir string, incremental bool) {
	t.Helper()
	cfg, err := config.Load(filepath.Join(inputDir, "config.yaml"))
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	cfg.Dirs.Content = filepath.Join(inputDir, cfg.Dirs.Content)
	cfg.Dirs.Theme = filepath.Join(inputDir, cfg.Dirs.Theme)
	cfg.Dirs.Static = filepath.Join(inputDir, cfg.Dirs.Static)
	cfg.Dirs.Output = outputDir

	parsers := map[string]core.Parser{"md": markdown.Parser()}
	err = pipeline.Build(context.Background(), cfg, parsers, nil, nil, pipeline.Options{
		Incremental: incremental,
		CacheDir:    cacheDir,
	})
	if err != nil {
		t.Fatalf("build (incremental=%v) failed: %v", incremental, err)
	}
}

func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	tree := map[string]string{}
	filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		b, _ := os.ReadFile(p)
		tree[rel] = string(b)
		return nil
	})
	return tree
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, path, extra string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	write(t, path, string(b)+extra)
}

func controlledConfig(name string) string {
	return "site:\n  name: \"" + name + "\"\n  base_url: \"\"\n" + `
parsers:
  note:
    rules:
      - match: '^!\s*(.+)$'
        emit:
          type: step
          content: "$1"

taxonomies:
  tags:
    path: "/tags/{term}"
    layout: "tag"
    index_path: "/tags"
    index_layout: "tag-index"

collections:
  notes:
    match: "notes/**"
    sort_by: "order"
    layout: "listing"
    path: "/notes"
    page_size: 2
`
}

func writeControlledSite(t *testing.T, dir string) {
	t.Helper()
	write(t, filepath.Join(dir, "config.yaml"), controlledConfig("Controlled Site"))

	write(t, filepath.Join(dir, "theme/base.html"), `<!DOCTYPE html>
<html><head><title>{{.Page.Envelope.title}} | {{.Site.Name}}</title></head>
<body>
{{partial "nav" .}}
{{block "content" .}}<main>{{.Page.Content}}</main>{{end}}
</body></html>
`)
	write(t, filepath.Join(dir, "theme/partials/nav.html"),
		`<nav>{{range sortBy (where .Site.Pages "type" "md") "url"}}<a href="{{.URL}}">{{.Envelope.title}}</a>{{end}}</nav>`)
	write(t, filepath.Join(dir, "theme/layouts/tag.html"), `{{define "content"}}<ul>{{.Page.Content}}</ul>{{end}}`)
	write(t, filepath.Join(dir, "theme/layouts/tag-index.html"), `{{define "content"}}<ul>{{.Page.Content}}</ul>{{end}}`)
	write(t, filepath.Join(dir, "theme/layouts/listing.html"),
		`{{define "content"}}<ol>{{.Page.Content}}</ol>{{if .Paginator}}<nav>page {{.Paginator.CurrentPage}}/{{.Paginator.TotalPages}}</nav>{{end}}{{end}}`)

	write(t, filepath.Join(dir, "content/index.md"), "---\ntitle: Home\n---\n\nWelcome.\n")
	write(t, filepath.Join(dir, "content/posts/alpha.md"), "---\ntitle: Alpha\ntags: [go, web]\n---\n\nAlpha body.\n")
	write(t, filepath.Join(dir, "content/posts/beta.md"), "---\ntitle: Beta\ntags: [go]\n---\n\nBeta body.\n")
	for i, n := range []string{"one", "two", "three"} {
		write(t, filepath.Join(dir, "content/notes/"+n+".note"),
			"---\ntitle: Note "+n+"\norder: "+string(rune('1'+i))+"\n---\n! step "+n+"\n")
	}
}
