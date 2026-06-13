package core

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseErrorFormat(t *testing.T) {
	pe := &ParseError{Line: 7, Err: fmt.Errorf("bad token")}
	if pe.Error() != "line 7: bad token" {
		t.Errorf("got %q", pe.Error())
	}

	noLine := &ParseError{Err: fmt.Errorf("bad token")}
	if noLine.Error() != "bad token" {
		t.Errorf("got %q", noLine.Error())
	}

	if !errors.Is(pe, pe.Err) {
		t.Error("ParseError should unwrap to its cause")
	}
}

func TestEngineErrorLineFormat(t *testing.T) {
	e := &EngineError{Phase: "PARSE", File: "content/a.md", Line: 3, Severity: LocalFatal, Err: fmt.Errorf("boom")}
	if got := e.Error(); got != "[PARSE] error content/a.md:3: boom" {
		t.Errorf("got %q", got)
	}
}

func TestYamlErrorFileLine(t *testing.T) {
	// yaml reports line 2 within the frontmatter block; the block's first
	// line sits at file line 2, so the file line is 3.
	if got := yamlErrorFileLine(errors.New("yaml: line 2: oops"), 2); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
	if got := yamlErrorFileLine(errors.New("no line info"), 2); got != 0 {
		t.Errorf("unmatched error should give 0, got %d", got)
	}
}

func TestSplitFrontmatterUnclosedReportsLine(t *testing.T) {
	// Two leading blank lines put the opening --- on file line 3.
	_, _, err := SplitFrontmatter([]byte("\n\n---\ntitle: x\n"))
	if err == nil {
		t.Fatal("expected unclosed frontmatter error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	if pe.Line != 3 {
		t.Errorf("unclosed frontmatter line = %d, want 3", pe.Line)
	}
}

func TestSplitFrontmatterYAMLErrorReportsFileLine(t *testing.T) {
	raw := []byte("---\ntitle: ok\n\tbad: tab-indent\n---\nbody\n")
	_, _, err := SplitFrontmatter(raw)
	if err == nil {
		t.Fatal("expected YAML error")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected ParseError, got %T", err)
	}
	// The offending line is file line 3; the exact value depends on yaml's
	// reporting, but it must be file-relative (greater than the yaml-relative
	// line, which is at most 2).
	if pe.Line < 3 {
		t.Errorf("expected file-relative line >= 3, got %d", pe.Line)
	}
}
