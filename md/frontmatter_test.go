package md

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/deck/config"
)

func TestFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		want     *Frontmatter
	}{
		{
			name: "with frontmatter",
			markdown: `---
title: Test Title
author: Test Author
tags:
  - tag1
  - tag2
---

# Slide Title

Content`,
			want: &Frontmatter{},
		},
		{
			name: "without frontmatter",
			markdown: `# Slide Title

Content`,
			want: nil,
		},
		{
			name:     "empty frontmatter",
			markdown: "---\n---\n\n# Slide Title",
			want:     &Frontmatter{},
		},
		{
			name: "frontmatter with trailing delimiter",
			markdown: `---
title: Test
---
# Slide Title`,
			want: &Frontmatter{},
		},
		{
			name: "frontmatter with any fields (all ignored)",
			markdown: `---
title: Test Title
author: Test Author
unknown_field: ignored
custom_data: 
  nested: value
metadata:
  key1: value1
  key2: value2
tags:
  - tag1
  - tag2
---
# Slide Title`,
			want: &Frontmatter{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := Parse(".", []byte(tt.markdown), nil)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if md == nil {
				t.Fatal("Parse() returned nil md")
				return
			}

			got := md.Frontmatter

			// Check if frontmatter matches expected value
			if tt.want == nil {
				if got != nil {
					t.Errorf("Parse() frontmatter = %+v, want nil", got)
				}
				return
			}

			if got == nil {
				t.Errorf("Parse() frontmatter = nil, want non-nil empty struct")
				return
			}

			// Since Frontmatter is an empty struct, just verify it's not nil when expected
			// All YAML fields are ignored, so no field comparison needed
		})
	}
}

func TestApplyFrontmatterToMD(t *testing.T) {
	tests := []struct {
		name            string
		initialContent  string
		title           string
		presentationID  string
		expectedContent string
	}{
		{
			name:           "new file without frontmatter",
			initialContent: "",
			title:          "Test Title",
			presentationID: "test-id-123",
			expectedContent: `---
presentationID: test-id-123
title: Test Title
---
`,
		},
		{
			name: "existing file without frontmatter",
			initialContent: `# My Presentation

This is the content.`,
			title:          "Test Title",
			presentationID: "test-id-123",
			expectedContent: `---
presentationID: test-id-123
title: Test Title
---
# My Presentation

This is the content.`,
		},
		{
			name: "existing file with frontmatter",
			initialContent: `---
author: John Doe
date: 2023-01-01
---
# My Presentation`,
			title:          "Updated Title",
			presentationID: "new-id-456",
			expectedContent: `---
author: John Doe
date: "2023-01-01"
presentationID: new-id-456
title: Updated Title
---
# My Presentation`,
		},
		{
			name: "existing file with frontmatter - update presentationID only",
			initialContent: `---
title: Existing Title
presentationID: old-id
---
# Content`,
			title:          "",
			presentationID: "new-id-789",
			expectedContent: `---
presentationID: new-id-789
title: Existing Title
---
# Content`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.md")

			// Write initial content if any
			if tt.initialContent != "" {
				if err := os.WriteFile(tmpFile, []byte(tt.initialContent), 0600); err != nil {
					t.Fatalf("Failed to write initial content: %v", err)
				}
			}

			// Run the function
			err := ApplyFrontmatterToMD(tmpFile, tt.title, tt.presentationID)
			if err != nil {
				t.Fatalf("ApplyFrontmatterToMD failed: %v", err)
			}

			// Read the result
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Fatalf("Failed to read result: %v", err)
			}

			// Compare normalized content (trim spaces and normalize line endings)
			got := strings.TrimSpace(string(content))
			want := strings.TrimSpace(tt.expectedContent)

			if got != want {
				t.Errorf("Content mismatch:\nGot:\n%s\n\nWant:\n%s", got, want)
			}
		})
	}
}

func TestApplyFrontmatterToMD_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "nested", "dir", "test.md")

	err := ApplyFrontmatterToMD(nestedFile, "Test", "test-id")
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(nestedFile); os.IsNotExist(err) {
		t.Errorf("File was not created: %s", nestedFile)
	}
}

func TestApplyConfig(t *testing.T) {
	tests := []struct {
		name               string
		initialFrontmatter *Frontmatter
		config             *config.Config
		want               *Frontmatter
	}{
		{
			name: "Apply config breaks when frontmatter breaks is not set",
			initialFrontmatter: &Frontmatter{
				Breaks: nil, // not set
			},
			config: &config.Config{
				Breaks: boolPtr(true),
			},
			want: &Frontmatter{
				Breaks:   boolPtr(true),
				Defaults: nil,
			},
		},
		{
			name: "Keep existing breaks value when already set",
			initialFrontmatter: &Frontmatter{
				Breaks: boolPtr(false), // already set
			},
			config: &config.Config{
				Breaks: boolPtr(true),
			},
			want: &Frontmatter{
				Breaks:   boolPtr(false), // keep existing value
				Defaults: nil,
			},
		},
		{
			name:               "Apply breaks when frontmatter is nil",
			initialFrontmatter: nil,
			config: &config.Config{
				Breaks: boolPtr(true),
			},
			want: &Frontmatter{
				Breaks:   boolPtr(true),
				Defaults: nil,
			},
		},
		{
			name: "Add config defaults when no existing defaults conditions",
			initialFrontmatter: &Frontmatter{
				Defaults: []DefaultCondition{}, // empty slice
			},
			config: &config.Config{
				Defaults: []config.DefaultCondition{
					{
						If:     "page == 1",
						Layout: "title",
						Freeze: boolPtr(true),
					},
				},
			},
			want: &Frontmatter{
				Breaks: nil,
				Defaults: []DefaultCondition{
					{
						If:     "page == 1",
						Layout: "title",
						Freeze: boolPtr(true),
					},
				},
			},
		},
		{
			name: "Append config defaults when existing defaults conditions present",
			initialFrontmatter: &Frontmatter{
				Defaults: []DefaultCondition{
					{
						If:     "page == 2",
						Layout: "content",
						Skip:   boolPtr(true),
					},
				},
			},
			config: &config.Config{
				Defaults: []config.DefaultCondition{
					{
						If:     "page == 1",
						Layout: "title",
						Freeze: boolPtr(true),
					},
					{
						If:     "page == 3",
						Layout: "end",
						Ignore: boolPtr(true),
					},
				},
			},
			want: &Frontmatter{
				Breaks: nil,
				Defaults: []DefaultCondition{
					{
						If:     "page == 2",
						Layout: "content",
						Skip:   boolPtr(true),
					},
					{
						If:     "page == 1",
						Layout: "title",
						Freeze: boolPtr(true),
					},
					{
						If:     "page == 3",
						Layout: "end",
						Ignore: boolPtr(true),
					},
				},
			},
		},
		{
			name: "Apply config codeBlockToImageCommand when frontmatter codeBlockToImageCommand is not set",
			initialFrontmatter: &Frontmatter{
				CodeBlockToImageCommand: "", // not set
			},
			config: &config.Config{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go",
			},
			want: &Frontmatter{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go",
				Defaults:                nil,
			},
		},
		{
			name: "Keep existing codeBlockToImageCommand value when already set",
			initialFrontmatter: &Frontmatter{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go", // already set
			},
			config: &config.Config{
				CodeBlockToImageCommand: "go run other/command",
			},
			want: &Frontmatter{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go", // keep existing value
				Defaults:                nil,
			},
		},
		{
			name:               "Apply codeBlockToImageCommand when frontmatter is nil",
			initialFrontmatter: nil,
			config: &config.Config{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go",
			},
			want: &Frontmatter{
				CodeBlockToImageCommand: "go run testdata/txt2img/main.go",
				Defaults:                nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.initialFrontmatter.applyConfig(tt.config)

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Frontmatter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
