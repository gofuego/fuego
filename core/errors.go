package core

import "fmt"

// Severity classifies the impact of an engine error.
type Severity int

const (
	Warning     Severity = iota // non-breaking, skip and continue
	LocalFatal                  // ruins one page, rest of site is valid
	GlobalFatal                 // compromises the entire build
)

func (s Severity) String() string {
	switch s {
	case Warning:
		return "warning"
	case LocalFatal:
		return "error"
	case GlobalFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// EngineError represents a structured error emitted by any pipeline phase.
type EngineError struct {
	Phase    string
	File     string
	Line     int
	Severity Severity
	Err      error
}

func (e *EngineError) Error() string {
	prefix := fmt.Sprintf("[%s] %s", e.Phase, e.Severity)
	if e.File != "" {
		prefix += fmt.Sprintf(" %s", e.File)
		if e.Line > 0 {
			prefix += fmt.Sprintf(":%d", e.Line)
		}
	}
	return fmt.Sprintf("%s: %s", prefix, e.Err)
}

func (e *EngineError) Unwrap() error {
	return e.Err
}

// ParseError attaches a source line number to a parser error. Parsers may
// return it (or wrap it) from Parse; the dispatcher unwraps it into
// EngineError.Line so build output points at file:line. Reporting positions
// is optional — plain errors keep working.
type ParseError struct {
	Line int
	Err  error
}

func (e *ParseError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("line %d: %s", e.Line, e.Err)
	}
	return e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}

// ErrorAccumulator collects engine errors during pipeline execution.
type ErrorAccumulator struct {
	errors []EngineError
}

func (a *ErrorAccumulator) Add(err EngineError) {
	a.errors = append(a.errors, err)
}

func (a *ErrorAccumulator) Errors() []EngineError {
	return a.errors
}

func (a *ErrorAccumulator) HasGlobalFatal() bool {
	for _, e := range a.errors {
		if e.Severity == GlobalFatal {
			return true
		}
	}
	return false
}

func (a *ErrorAccumulator) HasFatal() bool {
	for _, e := range a.errors {
		if e.Severity >= LocalFatal {
			return true
		}
	}
	return false
}
