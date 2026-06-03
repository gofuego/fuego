package core

import "testing"

func TestSplitFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEnv     bool
		wantPayload string
		wantErr     bool
	}{
		{
			name:        "standard frontmatter",
			input:       "---\ntitle: Hello\n---\nBody content",
			wantEnv:     true,
			wantPayload: "Body content",
		},
		{
			name:        "no frontmatter",
			input:       "Just plain text",
			wantEnv:     false,
			wantPayload: "Just plain text",
		},
		{
			name:    "unclosed frontmatter",
			input:   "---\ntitle: Hello\nBody content",
			wantErr: true,
		},
		{
			name:        "empty payload",
			input:       "---\ntitle: Hello\n---\n",
			wantEnv:     true,
			wantPayload: "",
		},
		{
			name:        "frontmatter with multiple fields",
			input:       "---\ntitle: Test\nlayout: quiz\ntags:\n  - go\n  - web\n---\nPayload here",
			wantEnv:     true,
			wantPayload: "Payload here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, payload, err := SplitFrontmatter([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantEnv && len(env) == 0 {
				t.Error("expected non-empty envelope")
			}
			if !tt.wantEnv && len(env) != 0 {
				t.Errorf("expected empty envelope, got %v", env)
			}

			if string(payload) != tt.wantPayload {
				t.Errorf("payload: got %q, want %q", string(payload), tt.wantPayload)
			}
		})
	}
}

func TestSplitFrontmatterFields(t *testing.T) {
	input := "---\ntitle: Hello World\nlayout: quiz\npoints: 10\n---\nbody"
	env, _, err := SplitFrontmatter([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["title"] != "Hello World" {
		t.Errorf("expected title 'Hello World', got %v", env["title"])
	}
	if env["layout"] != "quiz" {
		t.Errorf("expected layout 'quiz', got %v", env["layout"])
	}
	if env["points"] != 10 {
		t.Errorf("expected points 10, got %v", env["points"])
	}
}
