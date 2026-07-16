// Package dispatch resolves which registered parser claims a file, by its
// base name. It is the single source of truth for the claim rule shared by
// DISCOVER (content-vs-asset classification) and PARSE (parser dispatch), so
// a file classified as content is always parsed by the same parser that
// classified it.
//
// A parser claims files by exactly one kind: a parser that declares filename
// patterns (FilenameParser with a non-empty Filenames()) claims exactly those
// patterns; a parser without patterns claims its Type() as a bare extension.
// Patterns are the complete claim set — overriding a parser's patterns
// replaces its claims entirely (a markdown parser given the pattern README.md
// claims only README.md, not every .md file).
//
// The rule, in order:
//
//  1. Filename-pattern claims (a parser's FilenameParser.Filenames()) are
//     checked before bare-extension claims. A file like foo.adr.md is claimed
//     by the parser whose pattern *.adr.md matches, not by the "md" parser.
//  2. Among multiple matching patterns, the longest pattern string wins
//     (*.adr.md beats *.md beats *). Longer patterns are more specific.
//  3. Ties (equal-length patterns matching the same file) resolve by parser
//     precedence: the higher-precedence claim wins. Precedence is the order
//     the claims are supplied to NewResolver — later entries win — so callers
//     pass claims lowest-precedence first (declarative < earlier pack <
//     later pack < user-registered).
//  4. If no pattern matches, the bare extension is looked up. Extension claims
//     never tie: an extension maps to exactly one registered parser type.
//  5. No match → the file is an asset (Resolve returns matched=false).
package dispatch

import (
	"path/filepath"
	"strings"

	"github.com/gofuego/fuego/core"
)

// Resolver maps a file's base name to the parser type that claims it. Build it
// once per build from the registered parser set and reuse it across DISCOVER
// and PARSE. A Resolver is read-only after construction and safe for
// concurrent use.
type Resolver struct {
	// patterns is ordered by descending specificity: longest pattern first,
	// and within an equal length, higher precedence first. The first entry
	// whose pattern matches a base name wins.
	patterns []patternClaim
	// exts maps a bare extension (no leading dot) to its winning parser type.
	exts map[string]string
}

type patternClaim struct {
	pattern    string
	parserType string
	// prec is the precedence rank; higher wins a length tie.
	prec int
}

// NewResolver builds a resolver from parsers supplied in ascending precedence
// order (lowest precedence first, highest last). A parser that declares
// filename patterns (core.FilenameParser with a non-empty Filenames())
// contributes a pattern claim per entry and nothing else — its Type() is not
// implicitly claimed as an extension, so pattern claims are the parser's
// complete claim set. A parser without patterns contributes one extension
// claim keyed by Type(). When two parsers claim the same extension, the
// higher-precedence (later) one wins — matching the engine's existing
// "user > later pack > earlier pack > declarative" merge.
func NewResolver(parsersInPrecedenceOrder []core.Parser) *Resolver {
	r := &Resolver{exts: make(map[string]string)}

	for prec, p := range parsersInPrecedenceOrder {
		var pats []string
		if fp, ok := p.(core.FilenameParser); ok {
			pats = fp.Filenames()
		}
		if len(pats) == 0 {
			r.exts[p.Type()] = p.Type()
			continue
		}
		for _, pat := range pats {
			r.patterns = append(r.patterns, patternClaim{
				pattern:    pat,
				parserType: p.Type(),
				prec:       prec,
			})
		}
	}

	// Order patterns by descending specificity so Resolve can return the
	// first match: longest pattern first, then higher precedence first.
	sortPatternsBySpecificity(r.patterns)

	return r
}

// Resolve returns the parser type that claims baseName, and whether any parser
// claims it. baseName is a file's base name (e.g. "guide.adr.md"), not a full
// path — claims match base names only, never directory-scoped. When matched is
// false the file is an asset.
func (r *Resolver) Resolve(baseName string) (parserType string, matched bool) {
	// 1. Filename patterns, in descending specificity order.
	for _, c := range r.patterns {
		if ok, _ := filepath.Match(c.pattern, baseName); ok {
			return c.parserType, true
		}
	}

	// 2. Bare extension.
	ext := strings.TrimPrefix(filepath.Ext(baseName), ".")
	if ext != "" {
		if pt, ok := r.exts[ext]; ok {
			return pt, true
		}
	}

	return "", false
}

// sortPatternsBySpecificity orders claims most-specific first: longer pattern
// strings before shorter, and among equal lengths, higher precedence before
// lower. It is a stable insertion sort — the slice is tiny (one entry per
// registered filename pattern) and this keeps the ordering rule readable.
func sortPatternsBySpecificity(claims []patternClaim) {
	for i := 1; i < len(claims); i++ {
		c := claims[i]
		j := i - 1
		for j >= 0 && lessSpecific(claims[j], c) {
			claims[j+1] = claims[j]
			j--
		}
		claims[j+1] = c
	}
}

// lessSpecific reports whether a should sort after b (b is more specific).
// b is more specific when its pattern is longer, or equal length with higher
// precedence.
func lessSpecific(a, b patternClaim) bool {
	if len(a.pattern) != len(b.pattern) {
		return len(a.pattern) < len(b.pattern)
	}
	return a.prec < b.prec
}
