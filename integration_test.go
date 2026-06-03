package fuego_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FabioSol/fuego/core"
	"github.com/FabioSol/fuego/internal/config"
	"github.com/FabioSol/fuego/internal/pipeline"
	"github.com/FabioSol/fuego/parsers/markdown"
)

var update = flag.Bool("update", false, "update golden files")

// fixtureParserRegistry returns parsers to register for a given fixture name.
// The Markdown parser is registered for all fixtures since there are no
// built-in parsers. Additional compiled parsers are added per fixture.
func fixtureParserRegistry(fixtureName string) map[string]core.Parser {
	parsers := map[string]core.Parser{
		"md": markdown.Parser(),
	}

	switch fixtureName {
	case "compiled-parser", "declarative-compiled-collision", "comprehensive":
		parsers["card"] = &cardParser{}
	case "no-envelope":
		parsers["env"] = &envParser{}
	case "filename-parser":
		parsers["dockerfile"] = &dockerfileParser{}
	case "raw-node":
		parsers["raw"] = &rawPassthroughParser{}
	}

	return parsers
}

func TestIntegrationFixtures(t *testing.T) {
	fixtures, err := filepath.Glob("testdata/*")
	if err != nil {
		t.Fatalf("globbing fixtures: %v", err)
	}

	for _, fixture := range fixtures {
		info, err := os.Stat(fixture)
		if err != nil || !info.IsDir() {
			continue
		}

		t.Run(filepath.Base(fixture), func(t *testing.T) {
			t.Parallel()

			fixtureName := filepath.Base(fixture)
			inputDir := filepath.Join(fixture, "input")
			goldenDir := filepath.Join(fixture, "golden")
			outputDir := t.TempDir()

			cfgPath := filepath.Join(inputDir, "config.yaml")

			// Check if this is an error-case fixture
			expectedErrFile := filepath.Join(fixture, "expected_error")
			isErrorCase := false
			if _, errEx := os.Stat(expectedErrFile); errEx == nil {
				isErrorCase = true
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				if isErrorCase {
					checkExpectedError(t, err, expectedErrFile)
					return
				}
				t.Fatalf("loading config: %v", err)
			}

			cfg.Dirs.Content = filepath.Join(inputDir, cfg.Dirs.Content)
			cfg.Dirs.Theme = filepath.Join(inputDir, cfg.Dirs.Theme)
			cfg.Dirs.Static = filepath.Join(inputDir, cfg.Dirs.Static)
			cfg.Dirs.Output = outputDir

			parsers := fixtureParserRegistry(fixtureName)

			if isErrorCase {
				err := pipeline.Build(context.Background(), cfg, parsers, nil)
				if err == nil {
					t.Fatal("expected pipeline to fail, but it succeeded")
				}
				checkExpectedError(t, err, expectedErrFile)
				return
			}

			// Run pipeline
			err = pipeline.Build(context.Background(), cfg, parsers, nil)
			if err != nil {
				t.Fatalf("pipeline failed: %v", err)
			}

			if *update {
				os.RemoveAll(goldenDir)
				if err := copyDir(outputDir, goldenDir); err != nil {
					t.Fatalf("copying to golden: %v", err)
				}
				t.Logf("updated golden files for %s", fixtureName)
				return
			}

			compareDirectories(t, outputDir, goldenDir)
		})
	}
}

func checkExpectedError(t *testing.T, err error, expectedErrFile string) {
	t.Helper()
	expectedErr, _ := os.ReadFile(expectedErrFile)
	if expected := strings.TrimSpace(string(expectedErr)); expected != "" {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain expected %q", err.Error(), expected)
		}
	}
}

func compareDirectories(t *testing.T, actual, expected string) {
	t.Helper()

	err := filepath.WalkDir(expected, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		relPath, _ := filepath.Rel(expected, path)
		actualPath := filepath.Join(actual, relPath)

		expectedContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		actualContent, err := os.ReadFile(actualPath)
		if err != nil {
			t.Errorf("expected file %s not found in output", relPath)
			return nil
		}

		if string(actualContent) != string(expectedContent) {
			t.Errorf("file %s differs from golden:\n--- expected ---\n%s\n--- actual ---\n%s",
				relPath, string(expectedContent), string(actualContent))
		}

		return nil
	})
	if err != nil {
		t.Fatalf("walking expected dir: %v", err)
	}

	filepath.WalkDir(actual, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(actual, path)
		goldenPath := filepath.Join(expected, relPath)
		if _, err := os.Stat(goldenPath); os.IsNotExist(err) {
			t.Errorf("unexpected file in output: %s", relPath)
		}
		return nil
	})
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)
		dstPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(dstPath, 0755)
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dstPath, content, 0644)
	})
}

// --- CardParser: a compiled parser for .card flashcard files ---

type cardParser struct{}

func (p *cardParser) Type() string { return "card" }

func (p *cardParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	env, payload, err := core.SplitFrontmatter(raw)
	if err != nil {
		return nil, nil, err
	}
	if env == nil {
		env = make(core.Envelope)
	}

	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	var nodes []core.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if after, ok := strings.CutPrefix(line, "front:"); ok {
			nodes = append(nodes, core.Node{
				Type:    "front",
				Content: strings.TrimSpace(after),
			})
		} else if after, ok := strings.CutPrefix(line, "back:"); ok {
			nodes = append(nodes, core.Node{
				Type:    "back",
				Content: strings.TrimSpace(after),
			})
		} else {
			return nil, nil, fmt.Errorf("unrecognized card line: %q", line)
		}
	}

	return env, nodes, nil
}

// --- envParser: a no-envelope parser for .env files ---

type envParser struct{}

func (p *envParser) Type() string { return "env" }

func (p *envParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	env := core.Envelope{"title": "Environment Variables"}
	var nodes []core.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			nodes = append(nodes, core.Node{
				Type:       "env-var",
				Content:    parts[1],
				Attributes: map[string]any{"name": parts[0]},
			})
		}
	}

	return env, nodes, nil
}

// --- dockerfileParser: a filename-based parser for Dockerfile ---

type dockerfileParser struct{}

func (p *dockerfileParser) Type() string         { return "dockerfile" }
func (p *dockerfileParser) Filenames() []string   { return []string{"Dockerfile"} }

func (p *dockerfileParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	env := core.Envelope{"title": "Dockerfile"}
	var nodes []core.Node

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nodes = append(nodes, core.Node{
			Type:    "instruction",
			Content: line,
		})
	}

	return env, nodes, nil
}

// --- rawPassthroughParser: emits Raw nodes with a custom type ---

type rawPassthroughParser struct{}

func (p *rawPassthroughParser) Type() string { return "raw" }

func (p *rawPassthroughParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	env, payload, err := core.SplitFrontmatter(raw)
	if err != nil {
		return nil, nil, err
	}
	if env == nil {
		env = make(core.Envelope)
	}

	content := strings.TrimSpace(string(payload))
	if content == "" {
		return env, nil, nil
	}

	return env, []core.Node{
		{Type: "prerendered", Content: content, Raw: true},
	}, nil
}
