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

type Slides []*Page

type Config struct {
	Layout string `json:"layout,omitempty"` // layout name
	Freeze bool   `json:"freeze,omitempty"` // freeze the page
}

type Page struct {
	Layout    string   `json:"layout"`
	Freeze    bool     `json:"freeze,omitempty"`
	Titles    []string `json:"titles,omitempty"`
	Subtitles []string `json:"subtitles,omitempty"`
	Bodies    []*Body  `json:"bodies,omitempty"`
	Comments  []string `json:"comments,omitempty"`
}

type Body struct {
	Paragraphs []*Paragraph `json:"paragraphs,omitempty"`
}

type Paragraph struct {
	Fragments []*Fragment `json:"fragments,omitempty"`
	Bullet    Bullet      `json:"bullet,omitempty"`
	Nesting   int         `json:"nesting,omitempty"`
}

type Fragment struct {
	Value         string `json:"value"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Link          string `json:"link,omitempty"`
	SoftLineBreak bool   `json:"softLineBreak,omitempty"`
}

type Bullet string

const (
	BulletNone   Bullet = ""
	BulletDash   Bullet = "-"
	BulletNumber Bullet = "1"
	BulletAlpha  Bullet = "a"
)

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

// ParseFile
func ParseFile(f string) (Slides, error) {
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse
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
				nesting := v.Offset/2 - 1 // FIXME

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

func toFragments(b []byte, n ast.Node) ([]*Fragment, error) {
	var frags []*Fragment
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch n := c.(type) {
		case *ast.Emphasis:
			children, err := toFragments(b, n)
			if err != nil {
				return nil, err
			}
			frags = append(frags, &Fragment{
				Value:         children[0].Value,
				Bold:          (n.Level == 2),
				Italic:        (n.Level == 1),
				SoftLineBreak: children[0].SoftLineBreak,
			})
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
			frags = append(frags, &Fragment{
				Value:         convert(n.Text(b)),
				Bold:          false,
				SoftLineBreak: false,
			})
		}
	}
	return frags, nil
}

var convertRep = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n")

func convert(in []byte) string {
	return convertRep.Replace(string(in))
}
