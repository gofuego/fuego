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
	// MatchedParser is the parser type that matched this file by filename.
	// Empty when matched by extension.
	MatchedParser string
}

// FilenamePattern maps a filename pattern to the parser type that handles it.
type FilenamePattern struct {
	Pattern    string
	ParserType string
}

// Walk traverses the content directory and returns all discovered files.
// Files are categorized as content or asset based on their extension
// (checked against registeredTypes) or filename (checked against filenamePatterns).
func Walk(cfg *config.Config, registeredTypes map[string]bool, filenamePatterns []FilenamePattern) ([]FileEntry, error) {
	contentDir := cfg.Dirs.Content
	if !filepath.IsAbs(contentDir) {
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

		if ShouldIgnore(relPath, cfg.Ignore) {
			return nil
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		entry := FileEntry{
			Path:    path,
			RelPath: relPath,
			Ext:     ext,
			IsAsset: true,
		}

		if isContentExt(ext, registeredTypes) {
			entry.IsAsset = false
		} else if parserType, ok := matchFilename(filepath.Base(path), filenamePatterns); ok {
			entry.IsAsset = false
			entry.MatchedParser = parserType
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}

// isContentExt returns true if a parser is registered for this extension.
func isContentExt(ext string, registeredTypes map[string]bool) bool {
	return registeredTypes != nil && registeredTypes[ext]
}

// matchFilename checks if a filename matches any registered filename pattern.
func matchFilename(name string, patterns []FilenamePattern) (string, bool) {
	for _, p := range patterns {
		matched, _ := filepath.Match(p.Pattern, name)
		if matched {
			return p.ParserType, true
		}
	}
	return "", false
}
