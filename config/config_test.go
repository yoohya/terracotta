package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantError bool
		want      *Config
	}{
		{
			name:      "valid config",
			filename:  "valid.yaml",
			wantError: false,
			want: &Config{
				BasePath: "test/path",
				Modules: []Module{
					{Path: "module-a"},
					{Path: "module-b", DependsOn: []string{"module-a"}},
					{Path: "module-c", DependsOn: []string{"module-a", "module-b"}},
				},
			},
		},
		{
			name:      "no dependencies config",
			filename:  "no-deps.yaml",
			wantError: false,
			want: &Config{
				BasePath: "test/path",
				Modules: []Module{
					{Path: "module-a"},
					{Path: "module-b"},
					{Path: "module-c"},
				},
			},
		},
		{
			name:      "invalid yaml",
			filename:  "invalid.yaml",
			wantError: true,
			want:      nil,
		},
		{
			name:      "nonexistent file",
			filename:  "nonexistent.yaml",
			wantError: true,
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("..", "testdata", tt.filename)
			got, err := LoadConfig(path)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("LoadConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLoadConfigRelativePath(t *testing.T) {
	// Test that relative paths work correctly
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	testPath := filepath.Join(wd, "..", "testdata", "valid.yaml")
	cfg, err := LoadConfig(testPath)
	if err != nil {
		t.Errorf("failed to load config with relative path: %v", err)
	}
	if cfg == nil {
		t.Error("expected config to be non-nil")
	}
}
