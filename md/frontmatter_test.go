package md

import (
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
			presentation, err := Parse(".", []byte(tt.markdown))
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if presentation == nil {
				t.Fatal("Parse() returned nil presentation")
			}

			got := presentation.Frontmatter

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
