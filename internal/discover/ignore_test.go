package discover

import "testing"

func TestShouldIgnore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		relPath  string
		patterns []string
		want     bool
	}{
		{
			name:     "no patterns",
			relPath:  "hello.md",
			patterns: nil,
			want:     false,
		},
		{
			name:     "exact match",
			relPath:  ".DS_Store",
			patterns: []string{".DS_Store"},
			want:     true,
		},
		{
			name:     "globstar matches nested",
			relPath:  "deep/nested/.DS_Store",
			patterns: []string{"**/.DS_Store"},
			want:     true,
		},
		{
			name:     "globstar matches root",
			relPath:  ".DS_Store",
			patterns: []string{"**/.DS_Store"},
			want:     true,
		},
		{
			name:     "directory wildcard",
			relPath:  "blog/drafts/wip.md",
			patterns: []string{"**/drafts/*"},
			want:     true,
		},
		{
			name:     "non-matching pattern",
			relPath:  "blog/published/post.md",
			patterns: []string{"**/drafts/*"},
			want:     false,
		},
		{
			name:     "extension glob",
			relPath:  "images/photo.tmp",
			patterns: []string{"**/*.tmp"},
			want:     true,
		},
		{
			name:     "multiple patterns first matches",
			relPath:  "scratch.md",
			patterns: []string{"scratch.*", "**/.DS_Store"},
			want:     true,
		},
		{
			name:     "multiple patterns second matches",
			relPath:  "a/.DS_Store",
			patterns: []string{"scratch.*", "**/.DS_Store"},
			want:     true,
		},
		{
			name:     "multiple patterns none match",
			relPath:  "real/content.md",
			patterns: []string{"scratch.*", "**/.DS_Store"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ShouldIgnore(tt.relPath, tt.patterns)
			if got != tt.want {
				t.Errorf("ShouldIgnore(%q, %v) = %v, want %v", tt.relPath, tt.patterns, got, tt.want)
			}
		})
	}
}
