package route

import (
	"testing"

	"github.com/FabioSol/fuego/core"
)

func TestDetectCollisionsNormalizesTrailingSlash(t *testing.T) {
	// "/overview" and "/overview/" both write overview/index.html and must
	// collide even though the strings differ.
	pages := []*core.Page{
		{RelPath: "overview.md", URL: "/overview/"},
		{RelPath: "virtual:overview", URL: "/overview"},
	}

	errs := DetectCollisions(pages)
	if len(errs) != 1 {
		t.Fatalf("expected 1 collision, got %d", len(errs))
	}
	if errs[0].Severity != core.GlobalFatal {
		t.Errorf("collision severity = %v, want GlobalFatal", errs[0].Severity)
	}
}
