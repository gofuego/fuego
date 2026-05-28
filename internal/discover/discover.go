package discover

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/FabioSol/fuego/internal/config"
)

// FileEntry represents a discovered file in the content directory.
type FileEntry struct {
	// Path is the absolute path to the file.
	Path string
	// RelPath is the path relative to the content directory.
	RelPath string
	// Ext is the file extension without the leading dot (e.g., "card", "trivia").
	Ext string
	// IsAsset is true for binary/non-content files (images, PDFs, etc.)
	// that should be copied to output rather than parsed.
	IsAsset bool
}

// Known content file extensions that should never be treated as assets.
// This set is augmented at runtime by registered parser types.
var textExts = map[string]bool{
	"md": true, "markdown": true, "txt": true, "html": true, "htm": true,
}

// Walk traverses the content directory and returns all discovered files.
// Files are categorized as content or asset based on their extension.
// The registeredTypes parameter contains parser type names that should be
// treated as content extensions.
func Walk(cfg *config.Config, registeredTypes map[string]bool) ([]FileEntry, error) {
	contentDir := cfg.Dirs.Content
	if !filepath.IsAbs(contentDir) {
		// Content dir is relative to the working directory
		abs, err := filepath.Abs(contentDir)
		if err != nil {
			return nil, err
		}
		contentDir = abs
	}

	info, err := os.Stat(contentDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "walk", Path: contentDir, Err: os.ErrInvalid}
	}

	var entries []FileEntry

	err = filepath.WalkDir(contentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}

		// Apply ignore patterns
		if ShouldIgnore(relPath, cfg.Ignore) {
			return nil
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		isAsset := !isContentExt(ext, registeredTypes)

		entries = append(entries, FileEntry{
			Path:    path,
			RelPath: relPath,
			Ext:     ext,
			IsAsset: isAsset,
		})

		return nil
	})

	return entries, err
}

// isContentExt returns true if the extension indicates a parseable content file.
func isContentExt(ext string, registeredTypes map[string]bool) bool {
	if textExts[ext] {
		return true
	}
	if registeredTypes != nil && registeredTypes[ext] {
		return true
	}
	return false
}
