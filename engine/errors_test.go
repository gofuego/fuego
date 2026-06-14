package engine

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gofuego/fuego/core"
)

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  core.Severity
		want string
	}{
		{core.Warning, "warning"},
		{core.LocalFatal, "error"},
		{core.GlobalFatal, "fatal"},
		{core.Severity(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.sev.String(); got != tt.want {
			t.Errorf("Severity(%d).String() = %q, want %q", tt.sev, got, tt.want)
		}
	}
}

func TestEngineErrorImplementsError(t *testing.T) {
	var _ error = &core.EngineError{}
}

func TestEngineErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  core.EngineError
		want string
	}{
		{
			name: "with file and line",
			err: core.EngineError{
				Phase: "PARSE", File: "content/hello.card", Line: 14,
				Severity: core.LocalFatal, Err: fmt.Errorf("invalid frontmatter"),
			},
			want: "[PARSE] error content/hello.card:14: invalid frontmatter",
		},
		{
			name: "with file no line",
			err: core.EngineError{
				Phase: "ROUTE", File: "content/about.md",
				Severity: core.GlobalFatal, Err: fmt.Errorf("URL collision"),
			},
			want: "[ROUTE] fatal content/about.md: URL collision",
		},
		{
			name: "no file",
			err: core.EngineError{
				Phase: "INIT", Severity: core.GlobalFatal, Err: fmt.Errorf("config not found"),
			},
			want: "[INIT] fatal: config not found",
		},
		{
			name: "warning",
			err: core.EngineError{
				Phase: "DISCOVER", File: "content/notes.bak",
				Severity: core.Warning, Err: fmt.Errorf("no parser registered"),
			},
			want: "[DISCOVER] warning content/notes.bak: no parser registered",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestEngineErrorUnwrap(t *testing.T) {
	inner := fmt.Errorf("something broke")
	ee := &core.EngineError{Phase: "PARSE", Severity: core.LocalFatal, Err: inner}

	if !errors.Is(ee, inner) {
		t.Error("Unwrap() should expose the inner error")
	}
}

func TestErrorAccumulator(t *testing.T) {
	acc := &core.ErrorAccumulator{}

	if len(acc.Errors()) != 0 {
		t.Fatal("new accumulator should be empty")
	}
	if acc.HasFatal() {
		t.Fatal("empty accumulator should not have fatal")
	}
	if acc.HasGlobalFatal() {
		t.Fatal("empty accumulator should not have global fatal")
	}

	acc.Add(core.EngineError{Phase: "PARSE", Severity: core.Warning, Err: fmt.Errorf("warn")})
	if acc.HasFatal() {
		t.Error("warning should not count as fatal")
	}

	acc.Add(core.EngineError{Phase: "PARSE", Severity: core.LocalFatal, Err: fmt.Errorf("local")})
	if !acc.HasFatal() {
		t.Error("should have fatal after LocalFatal")
	}
	if acc.HasGlobalFatal() {
		t.Error("LocalFatal should not count as GlobalFatal")
	}

	acc.Add(core.EngineError{Phase: "ROUTE", Severity: core.GlobalFatal, Err: fmt.Errorf("collision")})
	if !acc.HasGlobalFatal() {
		t.Error("should have global fatal")
	}

	if len(acc.Errors()) != 3 {
		t.Errorf("expected 3 errors, got %d", len(acc.Errors()))
	}
}
