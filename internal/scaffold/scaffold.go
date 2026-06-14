package scaffold

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed all:templates
var templateFS embed.FS

// Data contains the template variables for scaffold generation.
type Data struct {
	Name   string
	Module string
	// PackImport is the module path of a format pack to install and wire in
	// (empty for no pack). PackSymbol is the package identifier used to call
	// its Pack() function.
	PackImport string
	PackSymbol string
}

// Generate creates a new Fuego project in the given directory and resolves
// its dependencies. It writes the project files (see WriteFiles) and then
// runs `go get`/`go mod tidy`.
func Generate(dir string, data Data) error {
	if err := WriteFiles(dir, data); err != nil {
		return err
	}
	resolveDeps(dir, data.PackImport)
	return nil
}

// appendPackNote records the installed pack in the project's CLAUDE.md so
// the project guide reflects what's wired in. The text is appended (not
// templated) to avoid clashing with the Go-template examples in CLAUDE.md.
func appendPackNote(dir string, data Data) error {
	note := fmt.Sprintf(`

## Installed format pack

This project was scaffolded with `+"`fuego init --pack %s`"+`. The pack is
wired in `+"`main.go`"+` via `+"`eng.Use(%s.Pack())`"+` and brings its own parser(s),
theme, config defaults, and hooks.

- Add the pack's content type under `+"`content/`"+` and build with `+"`go run . build`"+`.
- The pack contributes routes, taxonomies, and/or collections to your config.
  Run `+"`go run . config`"+` to see the fully resolved configuration with the
  provenance of each value (`+"`# user`"+` vs `+"`# pack: ...`"+`).
- Your `+"`theme/`"+` files override the pack's; delete a layout to fall back to
  the pack's version.
`, data.PackImport, data.PackSymbol)

	path := filepath.Join(dir, "CLAUDE.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("appending pack note to CLAUDE.md: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(note); err != nil {
		return fmt.Errorf("writing pack note: %w", err)
	}
	return nil
}

// WriteFiles renders the embedded scaffold templates into dir and writes
// go.mod. It performs no network access, so tests can generate a project
// and build it offline.
func WriteFiles(dir string, data Data) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	entries, err := walkEmbedFS("templates")
	if err != nil {
		return fmt.Errorf("reading templates: %w", err)
	}

	for _, entry := range entries {
		relPath := strings.TrimPrefix(entry, "templates/")
		dstPath := filepath.Join(dir, relPath)

		content, err := templateFS.ReadFile(entry)
		if err != nil {
			return fmt.Errorf("reading template %s: %w", entry, err)
		}

		// Handle .tmpl files — strip the extension and use Go's text/template
		if strings.HasSuffix(dstPath, ".tmpl") {
			dstPath = strings.TrimSuffix(dstPath, ".tmpl")
			rendered, err := renderTemplate(string(content), data)
			if err != nil {
				return fmt.Errorf("rendering template %s: %w", entry, err)
			}
			content = []byte(rendered)
		}

		// config.yaml also needs template rendering (for {{.Name}})
		if filepath.Base(dstPath) == "config.yaml" {
			rendered, err := renderTemplate(string(content), data)
			if err != nil {
				return fmt.Errorf("rendering config template: %w", err)
			}
			content = []byte(rendered)
		}

		if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dstPath, err)
		}
	}

	// Generate go.mod with just the module declaration
	goMod := fmt.Sprintf("module %s\n\ngo 1.23\n", data.Module)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0644); err != nil {
		return fmt.Errorf("writing go.mod: %w", err)
	}

	if data.PackImport != "" {
		if err := appendPackNote(dir, data); err != nil {
			return err
		}
	}
	return nil
}

// resolveDeps fetches the fuego dependency (and an optional format pack) and
// tidies the module. It only runs `go get`/`go mod tidy`, never `go run`/`go
// build`, so a pack's code is downloaded but never executed during init.
// Best effort — a failure prints guidance but does not abort scaffolding.
func resolveDeps(dir, packImport string) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return
	}

	mods := []string{"github.com/gofuego/fuego@latest"}
	if packImport != "" {
		mods = append(mods, packImport+"@latest")
	}
	for _, mod := range mods {
		getCmd := exec.Command(goPath, "get", mod)
		getCmd.Dir = dir
		getCmd.Stdout = os.Stdout
		getCmd.Stderr = os.Stderr
		if err := getCmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "fuego: warning: could not resolve %s — run 'go get %s' manually\n", mod, mod)
		}
	}

	tidyCmd := exec.Command(goPath, "mod", "tidy")
	tidyCmd.Dir = dir
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	// Best effort — don't fail init if tidy fails
	tidyCmd.Run()
}

func renderTemplate(content string, data Data) (string, error) {
	tmpl, err := template.New("scaffold").Parse(content)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// walkEmbedFS recursively lists all files in the embedded filesystem.
func walkEmbedFS(root string) ([]string, error) {
	var paths []string

	entries, err := templateFS.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fullPath := root + "/" + entry.Name()
		if entry.IsDir() {
			sub, err := walkEmbedFS(fullPath)
			if err != nil {
				return nil, err
			}
			paths = append(paths, sub...)
		} else {
			paths = append(paths, fullPath)
		}
	}

	return paths, nil
}
