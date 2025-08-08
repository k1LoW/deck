package md

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/deck/config"
	"github.com/tenntenn/golden"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in string
	}{
		{"../testdata/slide.md"},
		{"../testdata/cap.md"},
		{"../testdata/freeze.md"},
		{"../testdata/br.md"},
		{"../testdata/list_and_paragraph.md"},
		{"../testdata/paragraph_and_list.md"},
		{"../testdata/paragraphs.md"},
		{"../testdata/breaks_enabled.md"},
		{"../testdata/breaks_default.md"},
		{"../testdata/bold_and_italic.md"},
		{"../testdata/emoji.md"},
		{"../testdata/code.md"},
		{"../testdata/style.md"},
		{"../testdata/html_element_style.md"},
		{"../testdata/empty_list.md"},
		{"../testdata/empty_link.md"},
		{"../testdata/lists_with_blankline.md"},
		{"../testdata/nested_list.md"},
		{"../testdata/images.md"},
		{"../testdata/codeblock.md"},
		{"../testdata/frontmatter.md"},
		{"../testdata/autolink.md"},
		{"../testdata/heading.md"},
		{"../testdata/blockquote.md"},
		{"../testdata/ignore.md"},
		{"../testdata/skip.md"},
		{"../testdata/hr.md"},
		{"../testdata/tables.md"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			md, err := Parse("../testdata", b)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(md.Contents, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, "", tt.in, got)
				return
			}
			if diff := golden.Diff(t, "", tt.in, got); diff != "" {
				t.Error(diff)
			}
		})
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
			md := &MD{
				Frontmatter: tt.initialFrontmatter,
			}

			md.ApplyConfig(tt.config)

			if diff := cmp.Diff(tt.want, md.Frontmatter); diff != "" {
				t.Errorf("Frontmatter mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenCodeImage(t *testing.T) {
	ctx := context.Background()

	// Get absolute path to stub command
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	stubCmd := filepath.Join(cwd, "testdata", "stub_code_img.go")

	tests := []struct {
		name                string
		codeBlockToImageCmd string
		codeBlock           *CodeBlock
		wantErr             bool
	}{
		{
			name:                "output to stdout",
			codeBlockToImageCmd: "go run " + stubCmd,
			codeBlock:           &CodeBlock{},
		},
		{
			name:                "output to file using {{output}}",
			codeBlockToImageCmd: "go run " + stubCmd + " -o {{output}}",
			codeBlock:           &CodeBlock{},
		},
		{
			name:                "template expansion with {{lang}} and {{content}}",
			codeBlockToImageCmd: "go run " + stubCmd + " -o {{output}} && echo '{{lang}}: {{content}}' > /dev/null",
			codeBlock: &CodeBlock{
				Language: "javascript",
				Content:  "console.log('test')",
			},
		},
		{
			name:                "command failure",
			codeBlockToImageCmd: "false",
			codeBlock:           &CodeBlock{},
			wantErr:             true,
		},
		{
			name:                "invalid command",
			codeBlockToImageCmd: "this-command-does-not-exist",
			codeBlock:           &CodeBlock{},
			wantErr:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := genCodeImage(ctx, tt.codeBlockToImageCmd, tt.codeBlock)

			if (err != nil) != tt.wantErr {
				t.Errorf("genCodeImage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if img == nil {
					t.Error("genCodeImage() returned nil image without error")
					return
				}

				// Check that image data is not empty
				imgData := img.Bytes()
				pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
				if len(imgData) < len(pngHeader) {
					t.Error("genCodeImage() returned insufficient image data")
				}

				// For PNG images, verify basic structure
				if !reflect.DeepEqual(imgData[:len(pngHeader)], pngHeader) {
					t.Error("genCodeImage() did not return valid PNG data")
				}
			}
		})
	}
}

func TestToSlidesCodeBlockToImageCommand(t *testing.T) {
	ctx := context.Background()

	// Get absolute path to txt2img command
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	txt2imgDir := filepath.Join(cwd, "..", "testdata", "txt2img")
	txt2imgCmd := "cd " + txt2imgDir + " && go run main.go"

	tests := []struct {
		name                   string
		frontmatter            *Frontmatter
		codeBlockToImageCmdArg string
		hasCodeBlocks          bool
		expectError            bool
	}{
		{
			name: "Use argument command when both argument and frontmatter are provided",
			frontmatter: &Frontmatter{
				CodeBlockToImageCommand: txt2imgCmd,
			},
			codeBlockToImageCmdArg: txt2imgCmd,
			hasCodeBlocks:          true,
			expectError:            false,
		},
		{
			name: "Use frontmatter command when argument is empty",
			frontmatter: &Frontmatter{
				CodeBlockToImageCommand: txt2imgCmd,
			},
			codeBlockToImageCmdArg: "",
			hasCodeBlocks:          true,
			expectError:            false,
		},
		{
			name:                   "Use argument command when frontmatter is nil",
			frontmatter:            nil,
			codeBlockToImageCmdArg: txt2imgCmd,
			hasCodeBlocks:          true,
			expectError:            false,
		},
		{
			name: "No error when no code blocks present",
			frontmatter: &Frontmatter{
				CodeBlockToImageCommand: txt2imgCmd,
			},
			codeBlockToImageCmdArg: "",
			hasCodeBlocks:          false,
			expectError:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := &Content{}
			if tt.hasCodeBlocks {
				content.CodeBlocks = []*CodeBlock{
					{
						Language: "go",
						Content:  "fmt.Println(\"test\")",
					},
				}
			}

			md := &MD{
				Contents:    Contents{content},
				Frontmatter: tt.frontmatter,
			}

			_, err := md.ToSlides(ctx, tt.codeBlockToImageCmdArg)

			if tt.expectError && err == nil {
				t.Error("Expected error for non-existent command, got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// boolPtr is a helper function that returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`# Title

- A
- B

<br><br>

**C**
D
E<br>*F*

---

# Title

## Subtitle

- aA
- b**B**
- cC
    - dD
- *e*E
    - fF
        - gG
1. h**H**
  2. **i**I

ref: [deck repo](https://github.com/k1LoW/deck)

---

# Title
`))
	f.Fuzz(func(t *testing.T, in []byte) {
		_, _ = Parse(".", in)
	})
}
