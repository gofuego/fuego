package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioSol/fuego/internal/discover"
)

func TestCopyPublicDir(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create files in public/
	os.WriteFile(filepath.Join(srcDir, "favicon.ico"), []byte("icon-data"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "nested"), 0755)
	os.WriteFile(filepath.Join(srcDir, "nested", "robots.txt"), []byte("User-agent: *"), 0644)

	err := CopyPublicDir(srcDir, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check favicon at root
	data, err := os.ReadFile(filepath.Join(outDir, "favicon.ico"))
	if err != nil {
		t.Fatalf("favicon.ico not found: %v", err)
	}
	if string(data) != "icon-data" {
		t.Errorf("favicon.ico content mismatch: got %q", data)
	}

	// Check nested file
	data, err = os.ReadFile(filepath.Join(outDir, "nested", "robots.txt"))
	if err != nil {
		t.Fatalf("nested/robots.txt not found: %v", err)
	}
	if string(data) != "User-agent: *" {
		t.Errorf("robots.txt content mismatch: got %q", data)
	}
}

func TestCopyPublicDir_NonExistent(t *testing.T) {
	t.Parallel()

	err := CopyPublicDir("/nonexistent/path", t.TempDir())
	if err != nil {
		t.Fatalf("should not error on nonexistent public dir: %v", err)
	}
}

func TestCopyPublicDir_BrokenSymlink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	brokenLink := filepath.Join(dir, "public")
	os.Symlink(filepath.Join(dir, "nonexistent-target"), brokenLink)

	err := CopyPublicDir(brokenLink, t.TempDir())
	if err != nil {
		t.Fatalf("should not error on broken symlink: %v", err)
	}
}

func TestCopyAssets(t *testing.T) {
	t.Parallel()

	contentDir := t.TempDir()
	outDir := t.TempDir()

	// Create a colocated image
	imgDir := filepath.Join(contentDir, "blog", "images")
	os.MkdirAll(imgDir, 0755)
	imgPath := filepath.Join(imgDir, "photo.png")
	os.WriteFile(imgPath, []byte("png-data"), 0644)

	assets := []discover.FileEntry{
		{
			Path:    imgPath,
			RelPath: "blog/images/photo.png",
			Ext:     "png",
			IsAsset: true,
		},
	}

	err := CopyAssets(assets, contentDir, outDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "blog", "images", "photo.png"))
	if err != nil {
		t.Fatalf("asset not found in output: %v", err)
	}
	if string(data) != "png-data" {
		t.Errorf("asset content mismatch: got %q", data)
	}
}
