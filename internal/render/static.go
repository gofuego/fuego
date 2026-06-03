package render

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/FabioSol/fuego/internal/discover"
)

// CopyPublicDir copies the contents of the public/static directory verbatim
// to the output root. Files like favicon.ico, robots.txt, and _redirects
// are served at their original paths relative to the site root.
func CopyPublicDir(staticDir, outputDir string) error {
	info, err := os.Stat(staticDir)
	if err != nil {
		// Missing directory or broken symlink — both are fine, skip silently.
		if os.IsNotExist(err) {
			return nil
		}
		// Broken symlinks on some platforms may not match IsNotExist.
		// Fall back to Lstat: if the path itself doesn't exist, skip.
		if _, lErr := os.Lstat(staticDir); lErr != nil && os.IsNotExist(lErr) {
			return nil
		}
		return fmt.Errorf("checking static directory: %w", err)
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.WalkDir(staticDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(staticDir, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(outputDir, relPath)
		return copyFile(path, dstPath)
	})
}

// CopyAssets copies content-colocated binary assets (images, fonts, PDFs, etc.)
// to mirrored paths in the output directory, preserving directory structure.
func CopyAssets(assets []discover.FileEntry, contentDir, outputDir string) error {
	for _, asset := range assets {
		dstPath := filepath.Join(outputDir, filepath.FromSlash(asset.RelPath))
		if err := copyFile(asset.Path, dstPath); err != nil {
			return fmt.Errorf("copying asset %s: %w", asset.RelPath, err)
		}
	}
	return nil
}

// copyFile copies a single file from src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
