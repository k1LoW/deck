// Package md provides functionality for parsing markdown into slides.
package md

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/k1LoW/deck"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var allowedInlineHTMLElements = []string{
	"<a", "<abbr", "<b", "<cite", "<code", "<data", "<dfn", "<em", "<i", "<kbd",
	"<mark", "<q", "<rp", "<rt", "<ruby", "<s", "<samp", "<small", "<span",
	"<strong", "<sub", "<sup", "<time", "<u", "<var",
}

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
				if len(frags) == 0 {
					return ast.WalkContinue, nil
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
		if len(body.Paragraphs) > 0 && len(body.Paragraphs[0].Fragments) > 0 {
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
	var className string
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch childNode := c.(type) {
		case *ast.Emphasis:
			children, err := toFragments(b, childNode)
			if err != nil {
				return nil, err
			}
			for _, child := range children {
				frags = append(frags, &deck.Fragment{
					Value:         child.Value,
					Link:          child.Link,
					Bold:          (childNode.Level == 2) || child.Bold,
					Italic:        (childNode.Level == 1) || child.Italic,
					Code:          child.Code,
					SoftLineBreak: child.SoftLineBreak,
					ClassName:     className,
				})
			}
		case *ast.Link:
			children, err := toFragments(b, childNode)
			if err != nil {
				return nil, err
			}
			if len(children) == 0 {
				continue
			}
			frags = append(frags, &deck.Fragment{
				Value:         children[0].Value,
				Link:          convert(childNode.Destination),
				Bold:          children[0].Bold,
				Italic:        children[0].Italic,
				Code:          children[0].Code,
				SoftLineBreak: children[0].SoftLineBreak,
				ClassName:     className,
			})
		case *ast.Text:
			v := convert(childNode.Segment.Value(b))
			if v == "" {
				if len(frags) > 0 {
					frags[len(frags)-1].SoftLineBreak = childNode.SoftLineBreak()
				}
				continue // Skip empty text fragments
			}

			frags = append(frags, &deck.Fragment{
				Value:         convert(childNode.Segment.Value(b)),
				SoftLineBreak: childNode.SoftLineBreak(),
				ClassName:     className,
			})
		case *ast.RawHTML:
			// Get the raw HTML content
			htmlContent := string(childNode.Segments.Value(b))

			if !strings.HasPrefix(htmlContent, "<") {
				className = "" // Reset class attribute for closing tags
				continue       // Skip if it doesn't look like HTML
			}

			// Check if it's a closing tag
			if strings.HasPrefix(htmlContent, "</") && strings.HasSuffix(htmlContent, ">") {
				className = "" // Reset class attribute for closing tags
				continue
			}

			// <br> tag - add a newline fragment
			if strings.HasPrefix(htmlContent, "<br") {
				frags = append(frags, &deck.Fragment{
					Value:         "\n",
					Bold:          false,
					SoftLineBreak: false,
					ClassName:     className,
				})
				className = "" // Reset class attribute
				continue
			}

			// Check if the HTML content is an allowed inline element
			isAllowed := false
			for _, elem := range allowedInlineHTMLElements {
				if strings.HasPrefix(htmlContent, elem) {
					isAllowed = true
					break
				}
			}
			if !isAllowed {
				className = "" // Reset class attribute for disallowed elements
				continue       // Skip disallowed inline HTML elements
			}

			// Extract class attribute if present
			matches := classRe.FindStringSubmatch(htmlContent)
			if len(matches) > 1 {
				if matches[1] != "" {
					className = matches[1] // For double quotes
				} else if len(matches) > 2 && matches[2] != "" {
					className = matches[2] // For single quotes
				}
			}
		case *ast.String:
			// For String nodes, try to get their content
			if childNode.Value != nil {
				frags = append(frags, &deck.Fragment{
					Value:     convert(childNode.Value),
					ClassName: className,
				})
			} else {
				// Fallback for empty strings
				frags = append(frags, &deck.Fragment{
					Value: "",
				})
			}
		case *ast.CodeSpan:
			children, err := toFragments(b, childNode)
			if err != nil {
				return nil, err
			}
			frags = append(frags, &deck.Fragment{
				Value:         children[0].Value,
				Link:          children[0].Link,
				Bold:          children[0].Bold,
				Italic:        children[0].Italic,
				Code:          true,
				SoftLineBreak: children[0].SoftLineBreak,
				ClassName:     className,
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
	return frags, nil
}

// classRe is a regular expression to extract class attribute from HTML tags.
var classRe = regexp.MustCompile(`class="\s*([^"]*)\s*"|class='\s*([^']*)\s*'`)

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
	maxLen := max(oldLen, newLen)

	// Compare each page
	for i := range maxLen {
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
