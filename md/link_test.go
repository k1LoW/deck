package md

import (
	"testing"
)

func TestLinkWithUnderscore(t *testing.T) {
	// Test 1: Simple link with underscore
	markdown := "[link_text](https://github.com/k1LoW/deck)"
	md, err := Parse(".", []byte(markdown), nil)
	if err != nil {
		t.Fatal(err)
	}

	fragments := md.Contents[0].Bodies[0].Paragraphs[0].Fragments
	var gotText string
	for _, frag := range fragments {
		gotText += frag.Value
	}
	if gotText != "link_text" {
		t.Errorf("link text = %q, want %q", gotText, "link_text")
	}
}

func TestLinkWithUnderscoreAndFormatting(t *testing.T) {
	// Test 2: Link with bold text and underscore
	markdown := "[**bold_text**](https://github.com/k1LoW/deck)"
	md, err := Parse(".", []byte(markdown), nil)
	if err != nil {
		t.Fatal(err)
	}

	fragments := md.Contents[0].Bodies[0].Paragraphs[0].Fragments
	
	// Check all fragments are bold and have correct link
	var gotText string
	for _, frag := range fragments {
		gotText += frag.Value
		if !frag.Bold {
			t.Errorf("fragment %q should be bold", frag.Value)
		}
		if frag.Link != "https://github.com/k1LoW/deck" {
			t.Errorf("fragment link = %q, want %q", frag.Link, "https://github.com/k1LoW/deck")
		}
	}
	
	if gotText != "bold_text" {
		t.Errorf("link text = %q, want %q", gotText, "bold_text")
	}
}