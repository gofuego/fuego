package discover

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gofuego/fuego/internal/config"
	"github.com/gofuego/fuego/internal/dispatch"
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
	// MatchedParser is the parser type the dispatch resolver assigned to this
	// file. Empty for assets. PARSE dispatches by this exact value, so the
	// parser that classified the file as content is the one that parses it.
	MatchedParser string
}

// Walk traverses the content directory and returns all discovered files.
// Each file is classified as content or asset by the dispatch resolver, which
// applies the same claim rule PARSE uses: filename patterns before bare
// extensions, longest pattern wins, ties by parser precedence.
func Walk(cfg *config.Config, resolver *dispatch.Resolver) ([]FileEntry, error) {
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

		relPath, err := filepath.Rel(contentDir, path)
		if err != nil {
			return err
		}

		// Ignored directories are pruned (and never descended into), so a
		// pattern like ".git" or "node_modules" skips the whole subtree.
		if d.IsDir() {
			if relPath != "." && ShouldIgnore(relPath, cfg.Ignore) {
				return filepath.SkipDir
			}
			return nil
		}

		if ShouldIgnore(relPath, cfg.Ignore) {
			return nil
		}

		// Symlinks are reported by WalkDir as non-directory entries even when
		// they point at a directory, which would misclassify a symlinked dir as
		// a file asset and fail the copy. Skip symlinks: a shared artifact
		// reached through a symlink still renders at its canonical location.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		entry := FileEntry{
			Path:    path,
			RelPath: relPath,
			Ext:     ext,
			IsAsset: true,
		}

		if resolver != nil {
			if parserType, ok := resolver.Resolve(filepath.Base(path)); ok {
				entry.IsAsset = false
				entry.MatchedParser = parserType
			}
		}

		entries = append(entries, entry)
		return nil
	})

	return entries, err
}
