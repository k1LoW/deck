package md

import (
	"testing"
)

func TestLinkWithUnderscore(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		wantText string
		wantLink string
	}{
		{
			name:     "link with single underscore",
			markdown: "[link_text](https://example.com)",
			wantText: "link_text",
			wantLink: "https://example.com",
		},
		{
			name:     "link with multiple underscores",
			markdown: "[link_with_multiple_underscores](https://example.com)",
			wantText: "link_with_multiple_underscores",
			wantLink: "https://example.com",
		},
		{
			name:     "link with underscore at end",
			markdown: "[link_at_end_](https://example.com)",
			wantText: "link_at_end_",
			wantLink: "https://example.com",
		},
		{
			name:     "link with underscore at start",
			markdown: "[_link_at_start](https://example.com)",
			wantText: "_link_at_start",
			wantLink: "https://example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := Parse(".", []byte(tt.markdown), nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(md.Contents) == 0 {
				t.Fatal("no slides parsed")
			}
			
			if len(md.Contents[0].Bodies) == 0 {
				t.Fatal("no bodies in slide")
			}
			
			if len(md.Contents[0].Bodies[0].Paragraphs) == 0 {
				t.Fatal("no paragraphs in body")
			}
			
			fragments := md.Contents[0].Bodies[0].Paragraphs[0].Fragments
			if len(fragments) == 0 {
				t.Fatal("no fragments in paragraph")
			}
			
			gotText := fragments[0].Value
			gotLink := fragments[0].Link
			
			if gotText != tt.wantText {
				t.Errorf("link text = %q, want %q", gotText, tt.wantText)
			}
			if gotLink != tt.wantLink {
				t.Errorf("link URL = %q, want %q", gotLink, tt.wantLink)
			}
		})
	}
}