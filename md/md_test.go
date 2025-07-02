package md

import (
	"encoding/json"
	"os"
	"testing"

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
		{"../testdata/bold_and_italic.md"},
		{"../testdata/emoji.md"},
		{"../testdata/code.md"},
		{"../testdata/style.md"},
		{"../testdata/empty_list.md"},
		{"../testdata/empty_link.md"},
		{"../testdata/lists_with_blankline.md"},
		{"../testdata/nested_list.md"},
		{"../testdata/images.md"},
		{"../testdata/codeblock.md"},
		{"../testdata/frontmatter.md"},
		{"../testdata/autolink.md"},
		{"../testdata/heading.md"},
		{"../testdata/layout_by_heading.md"},
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
