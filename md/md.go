// Package md provides functionality for parsing markdown into slides.
package md

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"

	"github.com/k1LoW/deck"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Contents represents a collection of slide contents.
type Contents []*Content

// Config represents the configuration for a slide.
type Config struct {
	Layout string `json:"layout,omitempty"` // layout name
	Freeze bool   `json:"freeze,omitempty"` // freeze the page
}

// Content represents a single slide content.
type Content struct {
	Layout    string       `json:"layout"`
	Freeze    bool         `json:"freeze,omitempty"`
	Titles    []string     `json:"titles,omitempty"`
	Subtitles []string     `json:"subtitles,omitempty"`
	Bodies    []*deck.Body `json:"bodies,omitempty"`
	Comments  []string     `json:"comments,omitempty"`
}

// toBullet converts a marker byte to a Bullet type.
func toBullet(m byte) deck.Bullet {
	switch m {
	case '-', '+', '*':
		return deck.BulletDash
	case '.', ')':
		return deck.BulletNumber
	case 'a':
		return deck.BulletAlpha
	default:
		return deck.BulletNone
	}
}

// ParseFile parses a markdown file into contents.
func ParseFile(f string) (Contents, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse parses markdown bytes into contents.
// It splits the input by "---" delimiters and parses each section as a separate content.
func Parse(b []byte) (Contents, error) {
	bpages := bytes.Split(bytes.TrimPrefix(b, []byte("---\n")), []byte("\n---\n"))
	contents := make(Contents, len(bpages))
	for i, bpage := range bpages {
		content, err := ParseContent(bpage)
		if err != nil {
			return nil, err
		}
		contents[i] = content
	}

	return contents, nil
}

// ParseContent parses a single markdown content into a Content structure.
// It processes headings, lists, paragraphs, and HTML blocks to create a structured representation.
func ParseContent(b []byte) (*Content, error) {
	md := goldmark.New()
	reader := text.NewReader(b)
	doc := md.Parser().Parse(reader)
	content := &Content{}
	currentBody := &deck.Body{}
	content.Bodies = append(content.Bodies, currentBody)
	currentListMarker := deck.BulletNone
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Heading:
				switch v.Level {
				case 1:
					content.Titles = append(content.Titles, convert(v.Lines().Value(b)))
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &deck.Body{}
						content.Bodies = append(content.Bodies, currentBody)
					}
				case 2:
					content.Subtitles = append(content.Subtitles, convert(v.Lines().Value(b)))
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &deck.Body{}
						content.Bodies = append(content.Bodies, currentBody)
					}
				default:
					currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
						Fragments: []*deck.Fragment{
							{
								Value:         convert(v.Lines().Value(b)),
								Bold:          false,
								SoftLineBreak: false,
							},
						},
						Bullet:  deck.BulletNone,
						Nesting: 0,
					})
				}
			case *ast.List:
				currentListMarker = toBullet(v.Marker)
			case *ast.ListItem:
				tb := v.FirstChild()
				frags, err := toFragments(b, tb)
				if err != nil {
					return ast.WalkStop, err
				}
				// Calculate nesting level based on indentation
				// Assuming 2 spaces per indentation level and subtracting 1 for the base level
				nesting := 0
				if v.Offset >= 2 {
					nesting = v.Offset/2 - 1
				}

				currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
					Fragments: frags,
					Bullet:    currentListMarker,
					Nesting:   nesting,
				})
			case *ast.Paragraph:
				frags, err := toFragments(b, v)
				if err != nil {
					return ast.WalkStop, err
				}
				currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
					Fragments: frags,
					Bullet:    deck.BulletNone,
					Nesting:   0,
				})
			case *ast.HTMLBlock:
				if v.HTMLBlockType == ast.HTMLBlockType2 {
					block := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(convert(v.Lines().Value(b))), "<!--"), "-->"))
					config := &Config{}
					if err := json.Unmarshal([]byte(block), config); err == nil {
						content.Layout = config.Layout
						content.Freeze = config.Freeze
						return ast.WalkContinue, nil
					}
					content.Comments = append(content.Comments, block)
				} else {
					currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
						Fragments: []*deck.Fragment{
							{
								Value:         convert(bytes.Trim(v.Lines().Value(b), " \n")),
								Bold:          false,
								SoftLineBreak: false,
							},
						},
						Bullet:  deck.BulletNone,
						Nesting: 0,
					})
				}
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return nil, err
	}

	// remove empty bodies
	notEmpty := false
	for _, body := range content.Bodies {
		if len(body.Paragraphs) > 0 {
			notEmpty = true
			break
		}
	}
	if !notEmpty {
		content.Bodies = nil
	}

	return content, nil
}

// ToSlides converts the contents to a slice of deck.Slide structures.
func (contents Contents) ToSlides() deck.Slides {
	slides := make([]*deck.Slide, len(contents))
	for i, content := range contents {
		slides[i] = &deck.Slide{
			Layout:      content.Layout,
			Freeze:      content.Freeze,
			Titles:      content.Titles,
			Subtitles:   content.Subtitles,
			Bodies:      content.Bodies,
			SpeakerNote: strings.Join(content.Comments, "\n\n"),
		}
	}
	return slides
}

// toFragments converts an AST node to a slice of Fragment structures.
// It handles emphasis, links, text, and other node types to create formatted text fragments.
func toFragments(b []byte, n ast.Node) ([]*deck.Fragment, error) {
	var frags []*deck.Fragment
	if n == nil {
		return frags, nil
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Emphasis:
			children, err := toFragments(b, n)
			if err != nil {
				return nil, err
			}
			for _, child := range children {
				frags = append(frags, &deck.Fragment{
					Value:         child.Value,
					Bold:          (n.Level == 2) || child.Bold,
					Italic:        (n.Level == 1) || child.Italic,
					SoftLineBreak: child.SoftLineBreak,
				})
			}
		case *ast.Link:
			children, err := toFragments(b, n)
			if err != nil {
				return nil, err
			}
			if len(children) == 0 {
				continue
			}
			frags = append(frags, &deck.Fragment{
				Value:         children[0].Value,
				Link:          convert(n.Destination),
				Bold:          children[0].Bold,
				Italic:        children[0].Italic,
				SoftLineBreak: children[0].SoftLineBreak,
			})
		case *ast.Text:
			frags = append(frags, &deck.Fragment{
				Value:         convert(n.Segment.Value(b)),
				Bold:          false,
				SoftLineBreak: n.SoftLineBreak(),
			})
		default:
			// For line breaks and other special cases, we need to preserve the original behavior
			// Check the node type by its name
			switch typedNode := n.(type) {
			case *ast.RawHTML:
				// For RawHTML nodes (which include <br> tags), return a newline
				frags = append(frags, &deck.Fragment{
					Value:         "\n",
					Bold:          false,
					SoftLineBreak: false,
				})
			case *ast.String:
				// For String nodes, try to get their content
				if typedNode.Value != nil {
					frags = append(frags, &deck.Fragment{
						Value:         convert(typedNode.Value),
						Bold:          false,
						SoftLineBreak: false,
					})
				} else {
					// Fallback for empty strings
					frags = append(frags, &deck.Fragment{
						Value:         "",
						Bold:          false,
						SoftLineBreak: false,
					})
				}
			case *ast.CodeSpan:
				children, err := toFragments(b, typedNode)
				if err != nil {
					return nil, err
				}
				frags = append(frags, &deck.Fragment{
					Value:         children[0].Value,
					Bold:          children[0].Bold,
					Italic:        children[0].Italic,
					Code:          true,
					SoftLineBreak: children[0].SoftLineBreak,
				})
			default:
				// For all other node types, return a newline to match original behavior
				frags = append(frags, &deck.Fragment{
					Value:         "\n",
					Bold:          false,
					SoftLineBreak: false,
				})
			}
		}
	}
	return frags, nil
}

var convertRep = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n")

// convert transforms input bytes to a string, replacing HTML line break tags with newlines.
func convert(in []byte) string {
	return convertRep.Replace(string(in))
}

// DiffContents compares two Contents and returns the page numbers that have changed.
// Page numbers are 1-indexed.
func DiffContents(oldContents, newContents Contents) []int {
	var changedPages []int

	// Get the length of both Contents
	oldLen := len(oldContents)
	newLen := len(newContents)

	// Get the maximum length
	maxLen := oldLen
	if newLen > maxLen {
		maxLen = newLen
	}

	// Compare each page
	for i := 0; i < maxLen; i++ {
		// If a new page has been added
		if i >= oldLen {
			changedPages = append(changedPages, i+1) // 1-indexed
			continue
		}

		// If a page has been deleted
		if i >= newLen {
			// No action needed for deleted pages as they don't need to be applied
			continue
		}

		// Compare the content of the pages
		if !contentEqual(oldContents[i], newContents[i]) {
			if newContents[i].Freeze {
				// The frozen page is considered unchanged
				continue
			}
			changedPages = append(changedPages, i+1) // 1-indexed
		}
	}

	return changedPages
}

// contentEqual compares two Content structs and returns true if they are equal.
func contentEqual(old, new *Content) bool {
	if old == nil && new == nil {
		return true
	}
	if old == nil || new == nil {
		return false
	}

	// Compare layout and freeze flag
	if old.Layout != new.Layout || old.Freeze != new.Freeze {
		return false
	}

	// Compare titles
	if !reflect.DeepEqual(old.Titles, new.Titles) {
		return false
	}

	// Compare subtitles
	if !reflect.DeepEqual(old.Subtitles, new.Subtitles) {
		return false
	}

	// Compare comments
	if !reflect.DeepEqual(old.Comments, new.Comments) {
		return false
	}

	// Compare bodies
	if len(old.Bodies) != len(new.Bodies) {
		return false
	}

	for i, oldBody := range old.Bodies {
		newBody := new.Bodies[i]

		if len(oldBody.Paragraphs) != len(newBody.Paragraphs) {
			return false
		}

		for j, oldPara := range oldBody.Paragraphs {
			newPara := newBody.Paragraphs[j]

			if oldPara.Bullet != newPara.Bullet || oldPara.Nesting != newPara.Nesting {
				return false
			}

			if len(oldPara.Fragments) != len(newPara.Fragments) {
				return false
			}

			for k, oldFrag := range oldPara.Fragments {
				newFrag := newPara.Fragments[k]

				if oldFrag.Value != newFrag.Value ||
					oldFrag.Bold != newFrag.Bold ||
					oldFrag.Italic != newFrag.Italic ||
					oldFrag.Link != newFrag.Link ||
					oldFrag.SoftLineBreak != newFrag.SoftLineBreak {
					return false
				}
			}
		}
	}

	return true
}
