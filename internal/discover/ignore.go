package discover

import (
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// ShouldIgnore checks whether a file's relative path matches any of the ignore patterns.
// Patterns are evaluated using doublestar for ** (globstar) support.
// Paths are compared using forward slashes for cross-platform consistency.
func ShouldIgnore(relPath string, patterns []string) bool {
	// Normalize to forward slashes for consistent matching
	normalized := filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, normalized)
		if err != nil {
			continue // skip invalid patterns (already validated at config load)
		}
		if matched {
			return true
		}
	}
	return false
}
