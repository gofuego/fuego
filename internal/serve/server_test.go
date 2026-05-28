package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldProxy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want bool
	}{
		{"/assets/main.js", true},
		{"/assets/style.css", true},
		{"/@vite/client", true},
		{"/@fs/some/path", true},
		{"/node_modules/.vite/deps/react.js", true},
		{"/", false},
		{"/about/", false},
		{"/tags/go/", false},
		{"/favicon.ico", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			got := shouldProxy(tt.path)
			if got != tt.want {
				t.Errorf("shouldProxy(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestNewHandler_ServesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "about"), 0755)
	os.WriteFile(filepath.Join(dir, "about", "index.html"), []byte("<h1>About</h1>"), 0644)

	handler := NewHandler(dir, 0)

	req := httptest.NewRequest("GET", "/about/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "<h1>About</h1>" {
		t.Errorf("unexpected body: %q", rec.Body.String())
	}
}

func TestNewHandler_404ForMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	handler := NewHandler(dir, 0)

	req := httptest.NewRequest("GET", "/nonexistent/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}
