package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_SharedDrive(t *testing.T) {
	tests := []struct {
		name            string
		configYAML      string
		wantSharedDrive *bool
	}{
		{
			name: "config with sharedDrive true",
			configYAML: `
breaks: true
sharedDrive: true
defaults:
  - if: page == 1
    layout: title
`,
			wantSharedDrive: boolPtr(true),
		},
		{
			name: "config with sharedDrive false",
			configYAML: `
sharedDrive: false
`,
			wantSharedDrive: boolPtr(false),
		},
		{
			name: "config without sharedDrive",
			configYAML: `
breaks: true
`,
			wantSharedDrive: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for config
			tmpDir := t.TempDir()
			oldDataHome := os.Getenv("XDG_DATA_HOME")
			os.Setenv("XDG_DATA_HOME", tmpDir)
			defer os.Setenv("XDG_DATA_HOME", oldDataHome)

			// Reset dataHomePath
			dataHomePath = ""

			// Create deck config directory
			deckDir := filepath.Join(tmpDir, "deck")
			if err := os.MkdirAll(deckDir, 0755); err != nil {
				t.Fatalf("Failed to create deck directory: %v", err)
			}

			// Write config file
			configPath := filepath.Join(deckDir, "config.yml")
			if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Load config
			cfg, err := Load("")
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Check SharedDrive value
			if tt.wantSharedDrive == nil {
				if cfg.SharedDrive != nil {
					t.Errorf("SharedDrive = %v, want nil", *cfg.SharedDrive)
				}
			} else {
				if cfg.SharedDrive == nil {
					t.Errorf("SharedDrive = nil, want %v", *tt.wantSharedDrive)
				} else if *cfg.SharedDrive != *tt.wantSharedDrive {
					t.Errorf("SharedDrive = %v, want %v", *cfg.SharedDrive, *tt.wantSharedDrive)
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
