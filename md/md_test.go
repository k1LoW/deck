package md

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/tenntenn/golden"
)

var modtimeReg = regexp.MustCompile(`\n\s+"ModTime": ".*?",`)

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
			md, err := Parse("../testdata", b, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(md.Contents, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			// Remove ModTime for stable result
			got = modtimeReg.ReplaceAll(got, []byte(""))
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

func TestIsPageDelimiter(t *testing.T) {
	tests := []struct {
		name     string
		line     []byte
		expected bool
	}{
		// Basic valid delimiters
		{
			name:     "minimal three dashes",
			line:     []byte("---"),
			expected: true,
		},
		{
			name:     "four dashes",
			line:     []byte("----"),
			expected: true,
		},
		{
			name:     "five dashes",
			line:     []byte("-----"),
			expected: true,
		},
		{
			name:     "many dashes",
			line:     []byte("----------"),
			expected: true,
		},

		// Invalid cases - too short
		{
			name:     "empty line",
			line:     []byte(""),
			expected: false,
		},
		{
			name:     "one dash",
			line:     []byte("-"),
			expected: false,
		},
		{
			name:     "two dashes",
			line:     []byte("--"),
			expected: false,
		},

		// Invalid cases - contains non-dash characters
		{
			name:     "dashes with space in middle",
			line:     []byte("-- -"),
			expected: false,
		},
		{
			name:     "dashes with text in middle",
			line:     []byte("--a-"),
			expected: false,
		},
		{
			name:     "equals signs",
			line:     []byte("==="),
			expected: false,
		},
		{
			name:     "dashes with newline",
			line:     []byte("---\n"),
			expected: false,
		},

		// Trailing spaces and tabs
		{
			name:     "dashes with trailing space",
			line:     []byte("--- "),
			expected: true,
		},
		{
			name:     "dashes with multiple trailing spaces",
			line:     []byte("---  "),
			expected: true,
		},
		{
			name:     "dashes with trailing tabs",
			line:     []byte("---\t\t"),
			expected: true,
		},
		{
			name:     "dashes with trailing spaces and tabs",
			line:     []byte("---\t \t  "),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPageDelimiter(tt.line)
			if got != tt.expected {
				t.Errorf("isPageDelimiter(%q) = %v, want %v", string(tt.line), got, tt.expected)
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
		_, _ = Parse(".", in, nil)
	})
}

// TestCRLFSupport tests that CRLF line endings are handled correctly
// by comparing the parse results of LF and CRLF versions of the same file.
func TestCRLFSupport(t *testing.T) {
	// Parse LF version
	lfData, err := os.ReadFile("../testdata/slide.md")
	if err != nil {
		t.Fatalf("failed to read LF file: %v", err)
	}
	lfMD, err := Parse("../testdata", lfData, nil)
	if err != nil {
		t.Fatalf("failed to parse LF file: %v", err)
	}

	// Parse CRLF version
	crlfData, err := os.ReadFile("../testdata/slide.crlf.md")
	if err != nil {
		t.Fatalf("failed to read CRLF file: %v", err)
	}
	crlfMD, err := Parse("../testdata", crlfData, nil)
	if err != nil {
		t.Fatalf("failed to parse CRLF file: %v", err)
	}

	// Convert both to JSON for comparison
	lfJSON, err := json.MarshalIndent(lfMD.Contents, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal LF contents: %v", err)
	}

	crlfJSON, err := json.MarshalIndent(crlfMD.Contents, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal CRLF contents: %v", err)
	}

	// Compare the results
	if string(lfJSON) != string(crlfJSON) {
		t.Errorf("CRLF and LF versions produce different results.\nLF result:\n%s\n\nCRLF result:\n%s", string(lfJSON), string(crlfJSON))
	}

	// Also test ParseFile function for CRLF file
	crlfMDFromFile, err := ParseFile("../testdata/slide.crlf.md", nil)
	if err != nil {
		t.Fatalf("failed to parse CRLF file using ParseFile: %v", err)
	}

	crlfFromFileJSON, err := json.MarshalIndent(crlfMDFromFile.Contents, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal CRLF contents from ParseFile: %v", err)
	}

	if string(lfJSON) != string(crlfFromFileJSON) {
		t.Errorf("ParseFile with CRLF and Parse with LF produce different results.\nLF Parse result:\n%s\n\nCRLF ParseFile result:\n%s", string(lfJSON), string(crlfFromFileJSON))
	}
}
