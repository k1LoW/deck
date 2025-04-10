// Package md provides functionality for parsing markdown into slides.
package md

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Slides represents a collection of slide pages.
type Slides []*Page

// Config represents the configuration for a slide.
type Config struct {
	Layout string `json:"layout,omitempty"` // layout name
	Freeze bool   `json:"freeze,omitempty"` // freeze the page
}

// Page represents a single slide page.
type Page struct {
	Layout    string   `json:"layout"`
	Freeze    bool     `json:"freeze,omitempty"`
	Titles    []string `json:"titles,omitempty"`
	Subtitles []string `json:"subtitles,omitempty"`
	Bodies    []*Body  `json:"bodies,omitempty"`
	Comments  []string `json:"comments,omitempty"`
}

// Body represents the content body of a slide.
type Body struct {
	Paragraphs []*Paragraph `json:"paragraphs,omitempty"`
}

// Paragraph represents a paragraph within a slide body.
type Paragraph struct {
	Fragments []*Fragment `json:"fragments,omitempty"`
	Bullet    Bullet      `json:"bullet,omitempty"`
	Nesting   int         `json:"nesting,omitempty"`
}

// Fragment represents a text fragment within a paragraph.
type Fragment struct {
	Value         string `json:"value"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Link          string `json:"link,omitempty"`
	SoftLineBreak bool   `json:"softLineBreak,omitempty"`
}

// Bullet represents the type of bullet point for a paragraph.
type Bullet string

// Bullet constants for different bullet point types.
const (
	BulletNone   Bullet = ""
	BulletDash   Bullet = "-"
	BulletNumber Bullet = "1"
	BulletAlpha  Bullet = "a"
)

// toBullet converts a marker byte to a Bullet type.
func toBullet(m byte) Bullet {
	switch m {
	case '-', '+', '*':
		return BulletDash
	case '.', ')':
		return BulletNumber
	case 'a':
		return BulletAlpha
	default:
		return BulletNone
	}
}

// ParseFile parses a markdown file into slides.
func ParseFile(f string) (Slides, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse parses markdown bytes into slides.
// It splits the input by "---" delimiters and parses each section as a separate page.
func Parse(b []byte) (Slides, error) {
	bpages := bytes.Split(bytes.TrimPrefix(b, []byte("---\n")), []byte("\n---\n"))
	pages := make(Slides, len(bpages))
	for i, bpage := range bpages {
		page, err := ParsePage(bpage)
		if err != nil {
			return nil, err
		}
		pages[i] = page
	}

	return pages, nil
}

// ParsePage parses a single markdown page into a Page structure.
// It processes headings, lists, paragraphs, and HTML blocks to create a structured representation.
func ParsePage(b []byte) (*Page, error) {
	md := goldmark.New()
	reader := text.NewReader(b)
	doc := md.Parser().Parse(reader)
	page := &Page{}
	currentBody := &Body{}
	page.Bodies = append(page.Bodies, currentBody)
	currentListMarker := BulletNone
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Heading:
				switch v.Level {
				case 1:
					page.Titles = append(page.Titles, convert(v.Lines().Value(b)))
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &Body{}
						page.Bodies = append(page.Bodies, currentBody)
					}
				case 2:
					page.Subtitles = append(page.Subtitles, convert(v.Lines().Value(b)))
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &Body{}
						page.Bodies = append(page.Bodies, currentBody)
					}
				default:
					currentBody.Paragraphs = append(currentBody.Paragraphs, &Paragraph{
						Fragments: []*Fragment{
							{
								Value:         convert(v.Lines().Value(b)),
								Bold:          false,
								SoftLineBreak: false,
							},
						},
						Bullet:  BulletNone,
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

				currentBody.Paragraphs = append(currentBody.Paragraphs, &Paragraph{
					Fragments: frags,
					Bullet:    currentListMarker,
					Nesting:   nesting,
				})
			case *ast.Paragraph:
				frags, err := toFragments(b, v)
				if err != nil {
					return ast.WalkStop, err
				}
				currentBody.Paragraphs = append(currentBody.Paragraphs, &Paragraph{
					Fragments: frags,
					Bullet:    BulletNone,
					Nesting:   0,
				})
			case *ast.HTMLBlock:
				if v.HTMLBlockType == ast.HTMLBlockType2 {
					block := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(convert(v.Lines().Value(b))), "<!--"), "-->"))
					config := &Config{}
					if err := json.Unmarshal([]byte(block), config); err == nil {
						page.Layout = config.Layout
						page.Freeze = config.Freeze
						return ast.WalkContinue, nil
					}
					page.Comments = append(page.Comments, block)
				} else {
					currentBody.Paragraphs = append(currentBody.Paragraphs, &Paragraph{
						Fragments: []*Fragment{
							{
								Value:         convert(bytes.Trim(v.Lines().Value(b), " \n")),
								Bold:          false,
								SoftLineBreak: false,
							},
						},
						Bullet:  BulletNone,
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
	for _, body := range page.Bodies {
		if len(body.Paragraphs) > 0 {
			notEmpty = true
			break
		}
	}
	if !notEmpty {
		page.Bodies = nil
	}

	return page, nil
}

// toFragments converts an AST node to a slice of Fragment structures.
// It handles emphasis, links, text, and other node types to create formatted text fragments.
func toFragments(b []byte, n ast.Node) ([]*Fragment, error) {
	var frags []*Fragment
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Emphasis:
			children, err := toFragments(b, n)
			if err != nil {
				return nil, err
			}
			for _, child := range children {
				frags = append(frags, &Fragment{
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
			frags = append(frags, &Fragment{
				Value:         children[0].Value,
				Link:          convert(n.Destination),
				Bold:          children[0].Bold,
				SoftLineBreak: children[0].SoftLineBreak,
			})
		case *ast.Text:
			frags = append(frags, &Fragment{
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
				frags = append(frags, &Fragment{
					Value:         "\n",
					Bold:          false,
					SoftLineBreak: false,
				})
			case *ast.String:
				// For String nodes, try to get their content
				if typedNode.Value != nil {
					frags = append(frags, &Fragment{
						Value:         convert(typedNode.Value),
						Bold:          false,
						SoftLineBreak: false,
					})
				} else {
					// Fallback for empty strings
					frags = append(frags, &Fragment{
						Value:         "",
						Bold:          false,
						SoftLineBreak: false,
					})
				}
			default:
				// For all other node types, return a newline to match original behavior
				frags = append(frags, &Fragment{
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
