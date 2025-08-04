package md

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			want: &Frontmatter{Title: "Test Title"},
		},
		{
			name: "with sharedDrive true",
			markdown: `---
presentationID: test-123
title: Test Title
sharedDrive: true
---

# Slide Title`,
			want: &Frontmatter{
				PresentationID: "test-123",
				Title:          "Test Title",
				SharedDrive:    boolPtr(true),
			},
		},
		{
			name: "with sharedDrive false",
			markdown: `---
presentationID: test-456
sharedDrive: false
---

# Content`,
			want: &Frontmatter{
				PresentationID: "test-456",
				SharedDrive:    boolPtr(false),
			},
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
			want: &Frontmatter{Title: "Test"},
		},
		{
			name: "frontmatter with any fields",
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
			want: &Frontmatter{Title: "Test Title"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := Parse(".", []byte(tt.markdown))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if md == nil {
				t.Fatal("Parse() returned nil md")
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
				t.Errorf("Parse() frontmatter = nil, want non-nil")
				return
			}

			// Compare frontmatter fields
			if got.PresentationID != tt.want.PresentationID {
				t.Errorf("PresentationID = %q, want %q", got.PresentationID, tt.want.PresentationID)
			}
			if got.Title != tt.want.Title {
				t.Errorf("Title = %q, want %q", got.Title, tt.want.Title)
			}
			// Compare SharedDrive pointer values
			if tt.want.SharedDrive == nil {
				if got.SharedDrive != nil {
					t.Errorf("SharedDrive = %v, want nil", *got.SharedDrive)
				}
			} else {
				if got.SharedDrive == nil {
					t.Errorf("SharedDrive = nil, want %v", *tt.want.SharedDrive)
				} else if *got.SharedDrive != *tt.want.SharedDrive {
					t.Errorf("SharedDrive = %v, want %v", *got.SharedDrive, *tt.want.SharedDrive)
				}
			}
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
