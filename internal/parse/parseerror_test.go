package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/discover"
)

// lineReportingParser always fails with a positioned ParseError.
type lineReportingParser struct{}

func (p *lineReportingParser) Type() string { return "pos" }

func (p *lineReportingParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	return nil, nil, &core.ParseError{Line: 7, Err: fmt.Errorf("unexpected token")}
}

func TestParseErrorLineReachesEngineError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.pos")
	if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	file := discover.FileEntry{Path: path, RelPath: "broken.pos", Ext: "pos"}
	parsers := map[string]core.Parser{"pos": &lineReportingParser{}}

	_, engErr := parseFile(file, parsers)
	if engErr == nil {
		t.Fatal("expected parse error")
	}
	if engErr.Line != 7 {
		t.Errorf("EngineError.Line = %d, want 7", engErr.Line)
	}
	if !strings.Contains(engErr.Error(), "broken.pos:7") {
		t.Errorf("error %q should contain file:line", engErr.Error())
	}
}

func TestFrontmatterErrorLineInRawPassthrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.txt")
	if err := os.WriteFile(path, []byte("---\ntitle: x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// No parser for .txt: raw passthrough splits frontmatter and fails on the
	// unclosed block, which must carry the opening delimiter's line.
	file := discover.FileEntry{Path: path, RelPath: "bad.txt", Ext: "txt"}
	_, engErr := parseFile(file, map[string]core.Parser{})
	if engErr == nil {
		t.Fatal("expected unclosed frontmatter error")
	}
	if engErr.Line != 1 {
		t.Errorf("EngineError.Line = %d, want 1", engErr.Line)
	}
}
