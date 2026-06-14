package parse

import "github.com/gofuego/fuego/core"

// SplitFrontmatter delegates to core.SplitFrontmatter.
// Kept as a package-level function for internal callers during migration.
func SplitFrontmatter(raw []byte) (core.Envelope, []byte, error) {
	return core.SplitFrontmatter(raw)
}
