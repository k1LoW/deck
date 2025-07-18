// Package md provides functionality for parsing markdown into slides.
package md

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/exec"
	"github.com/k1LoW/expand"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/sync/errgroup"
)

var allowedInlineHTMLElements = []string{
	"a", "abbr", "b", "cite", "code", "data", "dfn", "em", "i", "kbd",
	"mark", "q", "rp", "rt", "ruby", "s", "samp", "small", "span",
	"strong", "sub", "sup", "time", "u", "var",
}

var allowdInlineElmReg *regexp.Regexp

func init() {
	// Compile the regular expression to match allowed inline HTML elements
	allowedElements := strings.Join(allowedInlineHTMLElements, "|")
	allowdInlineElmReg = regexp.MustCompile(`^<(` + allowedElements + `)[\s>]`)
}

// MD represents a markdown presentation.
type MD struct {
	Frontmatter *Frontmatter
	Contents    Contents
}

// Frontmatter represents YAML frontmatter data.
type Frontmatter struct {
	PresentationID string `yaml:"presentationID,omitempty" json:"presentationID,omitempty"` // ID of the Google Slides presentation
	Title          string `yaml:"title,omitempty" json:"title,omitempty"`                   // title of the presentation
	// Whether to display line breaks in the document as line breaks
	Breaks bool `yaml:"breaks,omitempty" json:"breaks,omitempty"`
}

// Contents represents a collection of slide contents.
type Contents []*Content

// Config represents the configuration for a slide.
type Config struct {
	Layout string `json:"layout,omitempty"` // layout name
	Freeze bool   `json:"freeze,omitempty"` // freeze the page
	Ignore bool   `json:"ignore,omitempty"` // ignore the page (skip slide generation)
	Skip   bool   `json:"skip,omitempty"`   // skip the page (do not show in the presentation)
}

type CodeBlock struct {
	Language string `json:"language,omitempty"`
	Content  string `json:"content"`
}

// Content represents a single slide content.
type Content struct {
	Layout         string             `json:"layout"`
	Freeze         bool               `json:"freeze,omitempty"`
	Ignore         bool               `json:"ignore,omitempty"`
	Skip           bool               `json:"skip,omitempty"`
	Titles         []string           `json:"titles,omitempty"`
	TitleBodies    []*deck.Body       `json:"-"`
	Subtitles      []string           `json:"subtitles,omitempty"`
	SubtitleBodies []*deck.Body       `json:"-"`
	Bodies         []*deck.Body       `json:"bodies,omitempty"`
	Images         []*deck.Image      `json:"images,omitempty"`
	CodeBlocks     []*CodeBlock       `json:"code_blocks,omitempty"`
	BlockQuotes    []*deck.BlockQuote `json:"block_quotes,omitempty"`
	Comments       []string           `json:"comments,omitempty"`
	Headings       map[int][]string   `json:"headings,omitempty"`
}

// ParseFile parses a markdown file into contents.
func ParseFile(f string) (_ *MD, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	abs, err := filepath.Abs(f)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(abs)
	return Parse(baseDir, b)
}

// Parse parses markdown bytes into contents.
// It splits the input by "---" delimiters and parses each section as a separate content.
func Parse(baseDir string, b []byte) (_ *MD, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	// Extract YAML frontmatter if present
	var frontmatter *Frontmatter
	mayHaveFrontmatter := bytes.HasPrefix(b, []byte("---\n"))
	bpages := splitPages(bytes.TrimPrefix(b, []byte("---\n")))

	if mayHaveFrontmatter && len(bpages) > 0 {
		maybeYAMLContent := bpages[0]
		frontmatter = &Frontmatter{}
		if err := yaml.Unmarshal(maybeYAMLContent, frontmatter); err == nil {
			bpages = bpages[1:] // Remove the first page if it contains frontmatter
		} else {
			frontmatter = nil
		}
	}
	var breaks bool
	if frontmatter != nil {
		breaks = frontmatter.Breaks
	}

	var contents Contents
	for _, bpage := range bpages {
		c, err := ParseContent(baseDir, bpage, breaks)
		if err != nil {
			return nil, err
		}
		// Skip ignored contents
		if !c.Ignore {
			contents = append(contents, c)
		}
	}

	return &MD{
		Frontmatter: frontmatter,
		Contents:    contents,
	}, nil
}

// ParseContent parses a single markdown content into a Content structure.
// It processes headings, lists, paragraphs, and HTML blocks to create a structured representation.
func ParseContent(baseDir string, b []byte, breaks bool) (_ *Content, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	// Parse once and reuse the AST
	md := goldmark.New()
	reader := text.NewReader(b)
	doc := md.Parser().Parse(reader)

	const sentinelLevel = 7 // H6 is the deepest level in HTML spec, so we use 7 as a sentinel value
	// First walk: determine title level
	titleLevel := sentinelLevel
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		// Only check on entering, skip on leaving
		if !entering {
			return ast.WalkContinue, nil
		}
		h, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}
		if h.Level < titleLevel {
			titleLevel = h.Level
			if titleLevel == 1 {
				return ast.WalkStop, nil
			}
		}
		// Skip children of headings since we only care about heading levels
		return ast.WalkSkipChildren, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to determine title level: %w", err)
	}

	// Second walk: parse content with determined title level
	content := &Content{
		Headings: make(map[int][]string),
	}
	if err := walkBodies(doc, baseDir, b, content, titleLevel, breaks); err != nil {
		return nil, fmt.Errorf("failed to walk body: %w", err)
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
func (contents Contents) ToSlides(ctx context.Context, codeBlockToImageCmd string) (_ deck.Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	slides := make([]*deck.Slide, len(contents))
	for i, content := range contents {
		var images []*deck.Image
		images = append(images, content.Images...)
		if codeBlockToImageCmd != "" && len(content.CodeBlocks) > 0 {
			mu := sync.Mutex{}
			eg := errgroup.Group{}
			blockMap := make(map[int]*deck.Image)
			for i, codeBlock := range content.CodeBlocks {
				eg.Go(func() error {
					image, err := genCodeImage(ctx, codeBlockToImageCmd, codeBlock)
					if err != nil {
						return err
					}
					mu.Lock()
					blockMap[i] = image
					mu.Unlock()
					return nil
				})
			}
			if err := eg.Wait(); err != nil {
				return nil, fmt.Errorf("failed to convert code blocks to images: %w", err)
			}
			for i := range content.CodeBlocks {
				images = append(images, blockMap[i])
			}
		}
		slides[i] = &deck.Slide{
			Layout:         content.Layout,
			Freeze:         content.Freeze,
			Skip:           content.Skip,
			Titles:         content.Titles,
			TitleBodies:    content.TitleBodies,
			Subtitles:      content.Subtitles,
			SubtitleBodies: content.SubtitleBodies,
			Bodies:         content.Bodies,
			Images:         images,
			BlockQuotes:    content.BlockQuotes,
			SpeakerNote:    strings.Join(content.Comments, "\n\n"),
		}
	}
	return slides, nil
}

func walkBodies(doc ast.Node, baseDir string, b []byte, content *Content, titleLevel int, breaks bool) error {
	currentBody := &deck.Body{}
	content.Bodies = append(content.Bodies, currentBody)
	currentListMarker := deck.BulletNone
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Heading:
				// TODO: apply inline styles to headings. ref. https://github.com/k1LoW/deck/issues/198
				var text string
				defaultFragment := deck.Fragment{}
				if v.Level > titleLevel+1 {
					defaultFragment.Bold = true
				}
				// don't support images in headings
				frags, _, err := toFragments(baseDir, b, v, defaultFragment)
				if err != nil {
					return ast.WalkStop, err
				}
				deckFrags := toDeckFragments(frags, breaks)
				for _, frag := range deckFrags {
					if frag.Value != "" {
						text += frag.Value
					}
				}
				content.Headings[v.Level] = append(content.Headings[v.Level], text)

				switch v.Level {
				case titleLevel:
					content.Titles = append(content.Titles, text)
					content.TitleBodies = append(content.TitleBodies, &deck.Body{
						Paragraphs: []*deck.Paragraph{{
							Fragments: deckFrags,
						}},
					})
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &deck.Body{}
						content.Bodies = append(content.Bodies, currentBody)
					}
				case titleLevel + 1:
					content.Subtitles = append(content.Subtitles, text)
					content.SubtitleBodies = append(content.SubtitleBodies, &deck.Body{
						Paragraphs: []*deck.Paragraph{{
							Fragments: deckFrags,
						}},
					})
					if len(currentBody.Paragraphs) > 0 {
						currentBody = &deck.Body{}
						content.Bodies = append(content.Bodies, currentBody)
					}
				default:
					currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
						Fragments: deckFrags,
						Bullet:    deck.BulletNone,
						Nesting:   0,
					})
				}
			case *ast.List:
				currentListMarker = toBullet(v.Marker)
			case *ast.ListItem:
				tb := v.FirstChild()
				frags, images, err := toFragments(baseDir, b, tb, deck.Fragment{})
				if err != nil {
					return ast.WalkStop, err
				}
				// Calculate nesting level based on indentation
				// Assuming 2 spaces per indentation level and subtracting 1 for the base level
				nesting := 0
				vv := v
				for {
					pv := vv.Parent()
					if pv == nil || pv.Kind() != ast.KindList {
						break
					}
					ppv := pv.Parent()
					if ppv != nil && ppv.Kind() == ast.KindListItem {
						nesting++
						listItem, ok := ppv.(*ast.ListItem)
						if !ok {
							break
						}
						vv = listItem
					} else {
						break
					}
				}
				content.Images = append(content.Images, images...)
				if len(frags) == 0 {
					return ast.WalkContinue, nil
				}
				currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
					Fragments: toDeckFragments(frags, breaks),
					Bullet:    currentListMarker,
					Nesting:   nesting,
				})
			case *ast.Paragraph:
				// Skip paragraphs that are direct children of list items to avoid duplication
				if v.Parent() != nil && v.Parent().Kind() == ast.KindListItem {
					return ast.WalkSkipChildren, nil
				}
				frags, images, err := toFragments(baseDir, b, v, deck.Fragment{})
				if err != nil {
					return ast.WalkStop, err
				}
				content.Images = append(content.Images, images...)
				if len(frags) == 0 {
					return ast.WalkContinue, nil
				}
				currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
					Fragments: toDeckFragments(frags, breaks),
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
						content.Ignore = config.Ignore
						content.Skip = config.Skip
						return ast.WalkContinue, nil
					}
					content.Comments = append(content.Comments, block)
				} else {
					currentBody.Paragraphs = append(currentBody.Paragraphs, &deck.Paragraph{
						Fragments: []*deck.Fragment{{
							Value: convert(bytes.Trim(v.Lines().Value(b), " \n")),
							Bold:  false,
						}},
						Bullet:  deck.BulletNone,
						Nesting: 0,
					})
				}
			case *ast.CodeBlock:
				c := v.Lines().Value(b)
				content.CodeBlocks = append(content.CodeBlocks, &CodeBlock{
					Content: string(c),
				})
			case *ast.FencedCodeBlock:
				lang := v.Language(b)
				c := v.Lines().Value(b)
				content.CodeBlocks = append(content.CodeBlocks, &CodeBlock{
					Language: string(lang),
					Content:  string(c),
				})
			case *ast.Blockquote:
				blockQuoteContent := &Content{}
				for v := n.FirstChild(); v != nil; v = v.NextSibling() {
					if err := walkBodies(v, baseDir, b, blockQuoteContent, 1, breaks); err != nil {
						return ast.WalkStop, err
					}
				}
				content.CodeBlocks = append(content.CodeBlocks, blockQuoteContent.CodeBlocks...)
				content.Images = append(content.Images, blockQuoteContent.Images...)
				for _, body := range blockQuoteContent.Bodies {
					if len(body.Paragraphs) > 0 {
						content.BlockQuotes = append(content.BlockQuotes, &deck.BlockQuote{
							Paragraphs: body.Paragraphs,
							Nesting:    0,
						})
					}
				}
				// Flatten nested block quotes
				for _, blockQuote := range blockQuoteContent.BlockQuotes {
					blockQuote.Nesting++
					content.BlockQuotes = append(content.BlockQuotes, blockQuote)
				}
				return ast.WalkSkipChildren, nil
			}
		}
		return ast.WalkContinue, nil
	}); err != nil {
		return err
	}
	return nil
}

func genCodeImage(ctx context.Context, codeBlockToImageCmd string, codeBlock *CodeBlock) (*deck.Image, error) {
	dir, err := os.MkdirTemp("", "deck")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(dir)

	output := filepath.Join(dir, "out.png")
	env := environToMap()
	env["CODEBLOCK_LANG"] = codeBlock.Language
	env["CODEBLOCK_CONTENT"] = codeBlock.Content
	env["CODEBLOCK_VALUE"] = codeBlock.Content // Deprecated, use CODEBLOCK_CONTENT.
	// I am unsure whether to set this as an environment variable, but I will set it for consistency.
	env["CODEBLOCK_OUTPUT"] = output
	store := map[string]any{
		"lang":    codeBlock.Language,
		"content": codeBlock.Content,
		"value":   codeBlock.Content, // Deprecated, use `content`.
		"output":  output,
		"env":     env,
	}
	repFn := expand.ExprRepFn("{{", "}}", store)
	replacedCmd, err := repFn(codeBlockToImageCmd)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "bash", "-c", replacedCmd)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(codeBlock.Content)
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run code block to image command: %w\nstdout: %s\nstderr: %s",
			err, stdout.String(), stderr.String())
	}
	// There is no need to check whether the output file path has been replaced in the template string.
	// It's sufficient to check whether the file exists after executing the command.
	// Furthermore, simply opening the file, rather than checking its existence,
	// also functions as a file existence check.
	b, err := os.ReadFile(output)
	if err != nil {
		b = stdout.Bytes() // use stdout if output file is not found
	}
	return deck.NewImageFromMarkdownBuffer(bytes.NewBuffer(b))
}

type fragment struct {
	*deck.Fragment
	SoftLineBreak bool
}

func toDeckFragments(frags []*fragment, breaks bool) []*deck.Fragment {
	deckFrags := make([]*deck.Fragment, len(frags))
	for i, frag := range frags {
		f := frag.Fragment
		if frag.SoftLineBreak && i < len(frags)-1 {
			// In the original Markdown and CommonMark specifications, SoftLineBreak between inline text
			// elements should be converted to a space character.
			// If breaks is true, it should be converted to a newline character.
			breakChar := " "
			if breaks {
				breakChar = "\n"
			}
			f.Value += breakChar
		}
		deckFrags[i] = f
	}
	return deckFrags
}

// toFragments converts an AST node to a slice of Fragment structures.
// It handles emphasis, links, text, and other node types to create formatted text fragments.
func toFragments(baseDir string, b []byte, n ast.Node, defaultFragment deck.Fragment) (_ []*fragment, _ []*deck.Image, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	var frags []*fragment
	var images []*deck.Image
	if n == nil {
		return frags, images, nil
	}
	var styleName string
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		switch childNode := c.(type) {
		case *ast.Emphasis:
			children, childImages, err := toFragments(baseDir, b, childNode, defaultFragment)
			if err != nil {
				return nil, nil, err
			}
			for _, child := range children {
				frags = append(frags, &fragment{
					SoftLineBreak: child.SoftLineBreak,
					Fragment: &deck.Fragment{
						Value:     child.Value,
						Link:      child.Link,
						Bold:      (childNode.Level == 2) || child.Bold,
						Italic:    (childNode.Level == 1) || child.Italic,
						Code:      child.Code,
						StyleName: styleName,
					}})
			}
			images = append(images, childImages...)
		case *ast.Link:
			children, childImages, err := toFragments(baseDir, b, childNode, defaultFragment)
			if err != nil {
				return nil, nil, err
			}
			if len(children) == 0 {
				continue
			}
			frags = append(frags, &fragment{
				SoftLineBreak: children[0].SoftLineBreak,
				Fragment: &deck.Fragment{
					Value:     children[0].Value,
					Link:      convert(childNode.Destination),
					Bold:      children[0].Bold,
					Italic:    children[0].Italic,
					Code:      children[0].Code,
					StyleName: styleName,
				}})
			images = append(images, childImages...)
		case *ast.AutoLink:
			url := string(childNode.URL(b))
			label := string(childNode.Label(b))
			frags = append(frags, &fragment{
				Fragment: &deck.Fragment{
					Value:     label,
					Link:      url,
					StyleName: styleName,
				}})
		case *ast.Text:
			v := convert(childNode.Segment.Value(b))
			if v == "" {
				if len(frags) > 0 {
					frags[len(frags)-1].SoftLineBreak = childNode.SoftLineBreak()
				}
				continue // Skip empty text fragments
			}
			value := convert(childNode.Segment.Value(b))
			if childNode.HardLineBreak() {
				value += "\n"
			}
			frag := defaultFragment
			frag.Value = value
			frag.StyleName = styleName
			frags = append(frags, &fragment{
				SoftLineBreak: childNode.SoftLineBreak(),
				Fragment:      &frag,
			})
		case *ast.Image:
			imageLink := string(childNode.Destination)
			if !strings.Contains(imageLink, "://") && !filepath.IsAbs(imageLink) {
				imageLink = filepath.Join(baseDir, imageLink)
			}
			image, err := deck.NewImageFromMarkdown(imageLink)
			if err != nil {
				return nil, nil, err
			}
			images = append(images, image)
		case *ast.RawHTML:
			// Get the raw HTML content
			htmlContent := string(childNode.Segments.Value(b))

			if !strings.HasPrefix(htmlContent, "<") {
				styleName = "" // Reset class attribute for closing tags
				continue       // Skip if it doesn't look like HTML
			}

			// Check if it's a closing tag
			if strings.HasPrefix(htmlContent, "</") && strings.HasSuffix(htmlContent, ">") {
				styleName = "" // Reset class attribute for closing tags
				continue
			}

			// <br> tag - add a newline fragment
			if strings.HasPrefix(htmlContent, "<br") {
				frags = append(frags, &fragment{
					Fragment: &deck.Fragment{
						Value:     "\n",
						Bold:      false,
						StyleName: styleName,
					}})
				styleName = "" // Reset class attribute
				continue
			}

			// Check if the HTML content is an allowed inline element
			stuffs := allowdInlineElmReg.FindStringSubmatch(htmlContent)
			isAllowed := len(stuffs) == 2
			if !isAllowed {
				styleName = "" // Reset class attribute for disallowed elements
				continue       // Skip disallowed inline HTML elements
			}

			// Extract class attribute if present
			matches := classRe.FindStringSubmatch(htmlContent)
			if len(matches) > 1 {
				if matches[1] != "" {
					styleName = matches[1] // For double quotes
				} else if len(matches) > 2 && matches[2] != "" {
					styleName = matches[2] // For single quotes
				}
			} else {
				styleName = stuffs[1] // Use the matched element name as style name
			}
		case *ast.String:
			// For String nodes, try to get their content
			if childNode.Value != nil {
				frag := defaultFragment
				frag.Value = convert(childNode.Value)
				frag.StyleName = styleName
				frags = append(frags, &fragment{Fragment: &frag})
			} else {
				// Fallback for empty strings
				frags = append(frags, &fragment{Fragment: &deck.Fragment{
					Value: "",
				}})
			}
		case *ast.CodeSpan:
			children, childImages, err := toFragments(baseDir, b, childNode, defaultFragment)
			if err != nil {
				return nil, nil, err
			}
			frags = append(frags, &fragment{
				SoftLineBreak: children[0].SoftLineBreak,
				Fragment: &deck.Fragment{
					Value:     children[0].Value,
					Link:      children[0].Link,
					Bold:      children[0].Bold,
					Italic:    children[0].Italic,
					Code:      true,
					StyleName: styleName,
				}})
			images = append(images, childImages...)
		default:
			// For all other node types, return a newline to match original behavior
			frags = append(frags, &fragment{Fragment: &deck.Fragment{
				Value: "\n",
				Bold:  false,
			}})
		}
	}
	return frags, images, nil
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
	if !slices.Equal(old.Titles, new.Titles) {
		return false
	}

	// Compare subtitles
	if !slices.Equal(old.Subtitles, new.Subtitles) {
		return false
	}

	// Compare comments
	if !slices.Equal(old.Comments, new.Comments) {
		return false
	}

	// Compare code blocks
	{
		a, err := json.Marshal(old.CodeBlocks)
		if err != nil {
			return false
		}
		b, err := json.Marshal(new.CodeBlocks)
		if err != nil {
			return false
		}
		if !bytes.Equal(a, b) {
			return false
		}
	}

	// Compare bodies
	{
		a, err := json.Marshal(old.Bodies)
		if err != nil {
			return false
		}
		b, err := json.Marshal(new.Bodies)
		if err != nil {
			return false
		}
		if !bytes.Equal(a, b) {
			return false
		}
	}

	// Compare images
	{
		if len(old.Images) != len(new.Images) {
			return false
		}
		var imageChecksums1 []uint32
		var imageChecksums2 []uint32
		for _, img := range old.Images {
			if img == nil {
				continue // Skip nil images
			}
			imageChecksums1 = append(imageChecksums1, img.Checksum())
		}
		for _, img := range new.Images {
			if img == nil {
				continue // Skip nil images
			}
			imageChecksums2 = append(imageChecksums2, img.Checksum())
		}
		slices.Sort(imageChecksums1)
		slices.Sort(imageChecksums2)
		if !slices.Equal(imageChecksums1, imageChecksums2) {
			return false
		}
	}

	return true
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

// splitPages splits markdown content by "---" delimiters
// while respecting code blocks (``` fenced blocks) to avoid splitting inside them.
func splitPages(b []byte) [][]byte {
	var (
		codeBlockMarker  = []byte("```")
		contentSeparator = []byte("---")
		newline          = []byte("\n")
	)

	lines := bytes.Split(b, newline)

	var pages [][]byte
	var currentPage [][]byte
	inCodeBlock := false

	for _, line := range lines {
		// Check if this line starts or ends a code block
		if bytes.HasPrefix(bytes.TrimSpace(line), codeBlockMarker) {
			inCodeBlock = !inCodeBlock
			currentPage = append(currentPage, line)
			continue
		}

		// Check if this is a page separator (only if not in code block)
		if !inCodeBlock && bytes.Equal(bytes.TrimSpace(line), contentSeparator) {
			// End current page and start a new one
			if len(currentPage) > 0 {
				pages = append(pages, bytes.Join(currentPage, newline))
				currentPage = nil
			}
			continue
		}

		// Add line to current page
		currentPage = append(currentPage, line)
	}

	// Add the last page if it has content
	if len(currentPage) > 0 {
		pages = append(pages, bytes.Join(currentPage, newline))
	}

	return pages
}

func environToMap() map[string]string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}
