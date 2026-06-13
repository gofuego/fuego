package fuego_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/FabioSol/fuego/parsers/markdown"
)

// BenchmarkBuild measures full pipeline builds against a deterministic,
// generated site: 70% Markdown pages, 30% declarative-DSL pages, taxonomies
// with a realistic term distribution, one collection, layouts and partials.
// Run with: go test -bench=Build -benchtime=3x .
func BenchmarkBuild(b *testing.B) {
	for _, bc := range []struct {
		pages   int
		hydrate bool // base template embeds {{.JSON}}
	}{
		{1000, true},
		{10000, true},
		{10000, false},
	} {
		name := fmt.Sprintf("%dpages", bc.pages)
		if !bc.hydrate {
			name += "-nojson"
		}
		b.Run(name, func(b *testing.B) {
			inputDir := b.TempDir()
			outputDir := b.TempDir()
			generateBenchSite(b, inputDir, bc.pages, bc.hydrate)

			cfg, err := config.Load(filepath.Join(inputDir, "config.yaml"))
			if err != nil {
				b.Fatalf("loading config: %v", err)
			}
			cfg.Dirs.Content = filepath.Join(inputDir, cfg.Dirs.Content)
			cfg.Dirs.Theme = filepath.Join(inputDir, cfg.Dirs.Theme)
			cfg.Dirs.Static = filepath.Join(inputDir, cfg.Dirs.Static)
			cfg.Dirs.Output = outputDir

			parsers := map[string]core.Parser{"md": markdown.Parser()}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := pipeline.Build(context.Background(), cfg, parsers, nil, nil); err != nil {
					b.Fatalf("pipeline failed: %v", err)
				}
			}
			b.StopTimer()

			perBuild := b.Elapsed().Seconds() / float64(b.N)
			b.ReportMetric(float64(bc.pages)/perBuild, "pages/sec")
			b.ReportMetric(perBuild*1000, "ms/build")
		})
	}
}

var benchTags = []string{"go", "ssg", "fuego", "templates", "parsers", "routing", "hooks", "themes"}

// generateBenchSite writes a deterministic synthetic site. Everything derives
// from a fixed seed so runs are comparable across machines and commits.
func generateBenchSite(tb testing.TB, dir string, pages int, hydrate bool) {
	tb.Helper()
	rng := rand.New(rand.NewSource(42))

	write := func(rel, content string) {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			tb.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			tb.Fatal(err)
		}
	}

	write("config.yaml", `site:
  name: "Bench Site"
  base_url: "https://bench.example.com"

parsers:
  note:
    rules:
      - match: '^!\s*(.+)$'
        emit:
          type: step
          content: "$1"
      - match: '^-\s*(.+)$'
        emit:
          type: detail
          content: "$1"

taxonomies:
  tags:
    path: "/tags/{term}"
    layout: "tag"
    index_path: "/tags"
    index_layout: "tag-index"

collections:
  guides:
    match: "guides/**"
    sort_by: "weight"
    layout: "listing"
    path: "/guides-index"
`)

	jsonEmbed := ""
	if hydrate {
		jsonEmbed = `<script type="application/json" id="fuego-data">{{.JSON}}</script>`
	}
	write("theme/base.html", fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><title>{{.Page.Envelope.title}} | {{.Site.Name}}</title></head>
<body>
{{partial "header" .}}
{{block "content" .}}<main>{{.Page.Content}}</main>{{end}}
%s
</body>
</html>
`, jsonEmbed))
	write("theme/partials/header.html", `<header><h1>{{.Site.Name}}</h1></header>`)
	write("theme/layouts/tag.html", `{{define "content"}}<ul class="tag">{{.Page.Content}}</ul>{{end}}`)
	write("theme/layouts/tag-index.html", `{{define "content"}}<ul class="tag-index">{{.Page.Content}}</ul>{{end}}`)
	write("theme/layouts/listing.html", `{{define "content"}}<ol class="listing">{{.Page.Content}}</ol>{{end}}`)
	write("theme/renderers/step.html", `<section class="step"><h3>{{.Content}}</h3></section>`)

	for i := 0; i < pages; i++ {
		// Zipf-ish tag assignment: low-numbered tags appear on far more pages.
		tagA := benchTags[rng.Intn(len(benchTags))]
		tagB := benchTags[rng.Intn(rng.Intn(len(benchTags))+1)]

		if i%10 < 7 {
			subdir := "posts"
			if i%5 == 0 {
				subdir = "guides"
			}
			write(fmt.Sprintf("content/%s/page-%05d.md", subdir, i), fmt.Sprintf(`---
title: "Page %05d"
weight: %d
tags:
  - %s
  - %s
---

# Page %05d

%s

## Details

%s

- item one of page %d
- item two of page %d
- item three of page %d

%s
`, i, i%50, tagA, tagB, i, benchParagraph(rng), benchParagraph(rng), i, i, i, benchParagraph(rng)))
		} else {
			var body string
			steps := 5 + rng.Intn(10)
			for s := 0; s < steps; s++ {
				body += fmt.Sprintf("! Step %d of note %d\n- detail %d-%d alpha\n- detail %d-%d beta\n", s, i, i, s, i, s)
			}
			write(fmt.Sprintf("content/notes/note-%05d.note", i), fmt.Sprintf(`---
title: "Note %05d"
weight: %d
tags:
  - %s
---
%s`, i, i%50, tagA, body))
		}
	}
}

func benchParagraph(rng *rand.Rand) string {
	words := []string{"fuego", "builds", "static", "sites", "from", "arbitrary", "content",
		"formats", "with", "deterministic", "output", "and", "parallel", "rendering",
		"across", "every", "pipeline", "phase", "including", "taxonomies"}
	n := 30 + rng.Intn(40)
	out := make([]byte, 0, n*8)
	for w := 0; w < n; w++ {
		if w > 0 {
			out = append(out, ' ')
		}
		out = append(out, words[rng.Intn(len(words))]...)
	}
	return string(out)
}
