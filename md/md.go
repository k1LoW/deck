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
	"github.com/google/cel-go/cel"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/exec"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"golang.org/x/sync/errgroup"
)

const sentinelLevel = 7 // H6 is the deepest level in HTML spec, so we use 7 as a sentinel value

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
	Breaks *bool `yaml:"breaks,omitempty" json:"breaks,omitempty"`
	// Conditions for default
	Defaults []DefaultCondition `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	// command to convert code blocks to images
	CodeBlockToImageCommand string `yaml:"codeBlockToImageCommand,omitempty" json:"codeBlockToImageCommand,omitempty"`
	// Variables for template substitution
	Variables map[string]string `yaml:"variables,omitempty" json:"variables,omitempty"`
}

type DefaultCondition struct {
	If     string `json:"if"`               // condition to check
	Layout string `json:"layout,omitempty"` // layout name to apply if condition is true
	Freeze *bool  `json:"freeze,omitempty"` // freeze the page
	Ignore *bool  `json:"ignore,omitempty"` // whether to ignore the page if condition is true
	Skip   *bool  `json:"skip,omitempty"`   // whether to skip the page if condition is true
}

// Contents represents a collection of slide contents.
type Contents []*Content

// Config represents the configuration for a slide.
type Config struct {
	Layout string `json:"layout,omitempty"` // layout name
	Freeze *bool  `json:"freeze,omitempty"` // freeze the page
	Ignore *bool  `json:"ignore,omitempty"` // ignore the page (skip slide generation)
	Skip   *bool  `json:"skip,omitempty"`   // skip the page (do not show in the presentation)
}

type CodeBlock struct {
	Language string `json:"language,omitempty"`
	Content  string `json:"content"`
}

// Content represents a single slide content.
type Content struct {
	Layout         string             `json:"layout"`
	Freeze         *bool              `json:"freeze,omitempty"`
	Ignore         *bool              `json:"ignore,omitempty"`
	Skip           *bool              `json:"skip,omitempty"`
	Titles         []string           `json:"titles,omitempty"`
	TitleBodies    []*deck.Body       `json:"-"`
	Subtitles      []string           `json:"subtitles,omitempty"`
	SubtitleBodies []*deck.Body       `json:"-"`
	Bodies         []*deck.Body       `json:"bodies,omitempty"`
	Images         []*deck.Image      `json:"images,omitempty"`
	CodeBlocks     []*CodeBlock       `json:"code_blocks,omitempty"`
	BlockQuotes    []*deck.BlockQuote `json:"block_quotes,omitempty"`
	Tables         []*deck.Table      `json:"tables,omitempty"`
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
	sep := []byte("---\n")

	// Extract YAML frontmatter if present
	var frontmatter *Frontmatter
	mayHaveFrontmatter := bytes.HasPrefix(b, sep)

	if mayHaveFrontmatter {
		stuff := bytes.SplitN(bytes.TrimPrefix(b, sep), sep, 2)
		if len(stuff) == 2 {
			frontmatter = &Frontmatter{}
			if err := yaml.Unmarshal(stuff[0], frontmatter); err == nil {
				b = stuff[1]
			} else {
				frontmatter = nil
			}
		}
	}
	bpages := splitPages(bytes.TrimPrefix(b, sep))
	var breaks bool
	if frontmatter != nil && frontmatter.Breaks != nil {
		breaks = *frontmatter.Breaks
	}

	var contents Contents
	for _, bpage := range bpages {
		c, err := ParseContent(baseDir, bpage, breaks)
		if err != nil {
			return nil, err
		}
		contents = append(contents, c)
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
	md := newParser()
	reader := text.NewReader(b)
	doc := md.Parser().Parse(reader)

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
	if err := walkContents(doc, baseDir, b, content, titleLevel, breaks); err != nil {
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

func (md *MD) ApplyConfig(cfg *config.Config) {
	if cfg == nil {
		return // No config to apply
	}
	if md.Frontmatter == nil {
		md.Frontmatter = &Frontmatter{}
	}
	if md.Frontmatter.Breaks == nil {
		md.Frontmatter.Breaks = cfg.Breaks
	}
	if md.Frontmatter.CodeBlockToImageCommand == "" {
		md.Frontmatter.CodeBlockToImageCommand = cfg.CodeBlockToImageCommand
	}
	// append default conditions from config
	for _, cond := range cfg.Defaults {
		md.Frontmatter.Defaults = append(md.Frontmatter.Defaults, DefaultCondition{
			If:     cond.If,
			Layout: cond.Layout,
			Freeze: cond.Freeze,
			Ignore: cond.Ignore,
			Skip:   cond.Skip,
		})
	}
}

func (md *MD) ToSlides(ctx context.Context, codeBlockToImageCmd string) (_ deck.Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	env, err := cel.NewEnv(
		cel.Variable("page", cel.IntType),
		cel.Variable("pageTotal", cel.IntType),
		cel.Variable("titles", cel.ListType(cel.StringType)),
		cel.Variable("subtitles", cel.ListType(cel.StringType)),
		cel.Variable("bodies", cel.ListType(cel.StringType)),
		cel.Variable("blockQuotes", cel.ListType(cel.StringType)),
		cel.Variable("codeBlocks", cel.ListType(cel.ObjectType("deck.CodeBlock"))),
		cel.Variable("images", cel.ListType(cel.ObjectType("deck.Image"))),
		cel.Variable("comments", cel.ListType(cel.StringType)),
		cel.Variable("headings", cel.MapType(cel.IntType, cel.ListType(cel.StringType))),
		cel.Variable("speakerNote", cel.StringType),
		cel.Variable("topHeadingLevel", cel.IntType),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}
	if md.Frontmatter == nil {
		return md.Contents.toSlides(ctx, codeBlockToImageCmd)
	}
	if codeBlockToImageCmd == "" {
		codeBlockToImageCmd = md.Frontmatter.CodeBlockToImageCommand
	}
	pageTotal := len(md.Contents)
	for i, content := range md.Contents {
		for _, cond := range md.Frontmatter.Defaults {
			ast, issues := env.Compile(fmt.Sprintf("!!(%s)", cond.If))
			if issues != nil && issues.Err() != nil {
				return nil, fmt.Errorf("failed to compile expression: %w", issues.Err())
			}
			prg, err := env.Program(ast)
			if err != nil {
				return nil, fmt.Errorf("failed to create program: %w", err)
			}
			var bodies []string
			for _, body := range content.Bodies {
				bodies = append(bodies, body.String())
			}
			var blockQuotes []string
			for _, blockQuote := range content.BlockQuotes {
				blockQuotes = append(blockQuotes, blockQuote.String())
			}
			for j := 1; j < sentinelLevel; j++ {
				if _, ok := content.Headings[j]; !ok {
					content.Headings[j] = []string{}
				}
			}
			var topHeadingLevel int
			for j := 1; j < sentinelLevel; j++ {
				if len(content.Headings[j]) > 0 {
					topHeadingLevel = j
					break
				}
			}
			out, _, err := prg.Eval(map[string]any{
				"page":            i + 1,
				"pageTotal":       pageTotal,
				"titles":          content.Titles,
				"subtitles":       content.Subtitles,
				"bodies":          bodies,
				"blockQuotes":     blockQuotes,
				"codeBlocks":      content.CodeBlocks,
				"images":          content.Images,
				"comments":        content.Comments,
				"headings":        content.Headings,
				"speakerNote":     strings.Join(content.Comments, "\n\n"),
				"topHeadingLevel": topHeadingLevel,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to evaluate values: %w", err)
			}
			if tf, ok := out.Value().(bool); ok && tf {
				if content.Layout == "" {
					content.Layout = cond.Layout
				}
				if content.Freeze == nil && cond.Freeze != nil {
					content.Freeze = cond.Freeze
				}
				if content.Ignore == nil && cond.Ignore != nil {
					content.Ignore = cond.Ignore
				}
				if content.Skip == nil && cond.Skip != nil {
					content.Skip = cond.Skip
				}
				break // Use the first matching condition
			}
		}
	}
	// Apply variable substitution if variables are defined
	if md.Frontmatter.Variables != nil && len(md.Frontmatter.Variables) > 0 {
		if err := md.applyVariableSubstitution(); err != nil {
			return nil, fmt.Errorf("failed to apply variable substitution: %w", err)
		}
	}
	return md.Contents.toSlides(ctx, codeBlockToImageCmd)
}

func newParser() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.Table),
	)
}

// toSlides converts the contents to a slice of deck.Slide structures.
func (contents Contents) toSlides(ctx context.Context, codeBlockToImageCmd string) (_ deck.Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	var slides []*deck.Slide
	for _, content := range contents {
		if content.Ignore != nil && *content.Ignore {
			// Skip ignored contents
			continue
		}
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
		slide := &deck.Slide{
			Layout:         content.Layout,
			Titles:         content.Titles,
			TitleBodies:    content.TitleBodies,
			Subtitles:      content.Subtitles,
			SubtitleBodies: content.SubtitleBodies,
			Bodies:         content.Bodies,
			Images:         images,
			BlockQuotes:    content.BlockQuotes,
			Tables:         content.Tables,
			SpeakerNote:    strings.Join(content.Comments, "\n\n"),
		}
		if content.Freeze != nil {
			slide.Freeze = *content.Freeze
		}
		if content.Skip != nil {
			slide.Skip = *content.Skip
		}
		slides = append(slides, slide)
	}
	return slides, nil
}

func walkContents(doc ast.Node, baseDir string, b []byte, content *Content, titleLevel int, breaks bool) error {
	currentBody := &deck.Body{}
	content.Bodies = append(content.Bodies, currentBody)
	currentListMarker := deck.BulletNone
	if err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch v := n.(type) {
			case *ast.Heading:
				var text string
				seedFragment := deck.Fragment{}
				if v.Level > titleLevel+1 {
					// Heading elements that are not at the level of titles or subtitles are embedded in the body,
					// so they are bold by default.
					seedFragment.Bold = true
				}
				// don't support images in headings for now
				frags, _, err := toFragments(baseDir, b, v, seedFragment)
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
			case *ast.ThematicBreak:
				if len(currentBody.Paragraphs) > 0 {
					currentBody = &deck.Body{}
					content.Bodies = append(content.Bodies, currentBody)
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
			case *east.Table:
				table, err := parseTable(v, baseDir, b, breaks)
				if err != nil {
					return ast.WalkStop, err
				}
				content.Tables = append(content.Tables, table)
				return ast.WalkSkipChildren, nil
			case *ast.Blockquote:
				blockQuoteContent := &Content{
					Headings: make(map[int][]string),
				}
				for v := n.FirstChild(); v != nil; v = v.NextSibling() {
					if err := walkContents(v, baseDir, b, blockQuoteContent, 1, breaks); err != nil {
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
	replacedCmd, err := expandTemplate(codeBlockToImageCmd, store)
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
	return deck.NewImageFromCodeBlock(bytes.NewBuffer(b))
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
func toFragments(baseDir string, b []byte, n ast.Node, seedFragment deck.Fragment) (_ []*fragment, _ []*deck.Image, err error) {
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
			children, childImages, err := toFragments(baseDir, b, childNode, seedFragment)
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
			children, childImages, err := toFragments(baseDir, b, childNode, seedFragment)
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
			if childNode.HardLineBreak() {
				v += "\n"
			}
			frag := seedFragment
			frag.Value = v
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
				frag := seedFragment
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
			children, childImages, err := toFragments(baseDir, b, childNode, seedFragment)
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
			if newContents[i].Freeze != nil && *newContents[i].Freeze {
				// The frozen page is considered unchanged
				continue
			}
			changedPages = append(changedPages, i+1) // 1-indexed
		}
	}

	return changedPages
}

func jsonEqual[T any](a, b T) bool {
	aBytes, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bBytes, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(aBytes, bBytes)
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
	if !jsonEqual(old.CodeBlocks, new.CodeBlocks) {
		return false
	}

	// Compare bodies
	if len(old.Bodies) != len(new.Bodies) {
		return false
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

	// Compare block quotes
	if !jsonEqual(old.BlockQuotes, new.BlockQuotes) {
		return false
	}

	// Compare tables
	if !jsonEqual(old.Tables, new.Tables) {
		return false
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

func isPageDelimiter(line []byte) bool {
	return len(line) >= 3 && !slices.ContainsFunc(line, func(b byte) bool {
		return b != '-'
	})
}

// splitPages splits markdown content by delimiters
// while respecting fenced code blocks and setext headings to avoid splitting inside them.
func splitPages(b []byte) [][]byte {
	md := newParser()
	reader := text.NewReader(b)
	doc := md.Parser().Parse(reader)

	// Count original thematic breaks
	originalBreakCount := 0
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if _, ok := n.(*ast.ThematicBreak); ok {
				originalBreakCount++
			}
		}
		return ast.WalkContinue, nil
	})

	lines := bytes.Split(b, []byte("\n"))
	var separatorLines = []int{-1} // Start with -1 to handle the first page correctly
	// For each potential delimiter line, check if removing it would reduce the thematic break count
	for lineNum, line := range lines {
		if isPageDelimiter(line) {
			// Create content with this line replaced by a space
			modifiedLines := make([][]byte, len(lines))
			copy(modifiedLines, lines)
			modifiedLines[lineNum] = []byte(" ") // Replace with space to maintain structure
			modifiedContent := bytes.Join(modifiedLines, []byte("\n"))

			// Parse the modified content
			modifiedDoc := md.Parser().Parse(text.NewReader(modifiedContent))

			// Count thematic breaks in modified content
			modifiedBreakCount := 0
			_ = ast.Walk(modifiedDoc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
				if entering {
					if _, ok := n.(*ast.ThematicBreak); ok {
						modifiedBreakCount++
					}
				}
				return ast.WalkContinue, nil
			})

			// If removing this line reduces the thematic break count, it's a real separator
			if modifiedBreakCount == originalBreakCount-1 {
				separatorLines = append(separatorLines, lineNum)
			}
		}
	}

	// Split content based on separator line positions
	var bpages [][]byte
	for i, sepLine := range separatorLines {
		from := sepLine + 1
		to := len(lines)
		if i < len(separatorLines)-1 {
			to = separatorLines[i+1]
		}
		pageLines := lines[from:to]
		pageContent := bytes.TrimSpace(bytes.Join(pageLines, []byte("\n")))
		if len(pageContent) > 0 {
			bpages = append(bpages, pageContent)
		}
	}
	return bpages
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

// Regular expression to match {{expression}} patterns.
var celExprReg = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// expandTemplate expands template expressions in the format {{CEL expression}} with values from the store.
// It supports CEL (Common Expression Language) expressions within the template.
func expandTemplate(template string, store map[string]any) (string, error) {
	// Create CEL environment with store variables
	env, err := createCELEnv(store)
	if err != nil {
		return "", fmt.Errorf("failed to create CEL environment: %w", err)
	}

	var expandErr error
	result := celExprReg.ReplaceAllStringFunc(template, func(match string) string {
		// Extract CEL expression without {{ }}
		expr := strings.TrimSpace(match[2 : len(match)-2])

		// Compile and evaluate CEL expression
		ast, issues := env.Compile(expr)
		if issues != nil && issues.Err() != nil {
			expandErr = fmt.Errorf("template compilation error for '{{%s}}': %w", expr, issues.Err())
			return match // Return original match on error
		}

		prg, err := env.Program(ast)
		if err != nil {
			expandErr = fmt.Errorf("template program creation error for '{{%s}}': %w", expr, err)
			return match // Return original match on error
		}

		out, _, err := prg.Eval(store)
		if err != nil {
			expandErr = fmt.Errorf("template evaluation error for '{{%s}}': %w", expr, err)
			return match // Return original match on error
		}

		// Convert result to string
		return fmt.Sprintf("%v", out.Value())
	})

	if expandErr != nil {
		return "", expandErr
	}

	return result, nil
}

// createCELEnv creates a CEL environment with all variables from the store.
func createCELEnv(store map[string]any) (*cel.Env, error) {
	var options []cel.EnvOption

	// Add each top-level store key as a CEL variable
	for key, value := range store {
		celType := inferCELType(value)
		options = append(options, cel.Variable(key, celType))
	}

	return cel.NewEnv(options...)
}

// inferCELType infers the CEL type from a Go value.
func inferCELType(value any) *cel.Type {
	switch value.(type) {
	case string:
		return cel.StringType
	case int, int32, int64:
		return cel.IntType
	case float32, float64:
		return cel.DoubleType
	case bool:
		return cel.BoolType
	case map[string]any:
		return cel.MapType(cel.StringType, cel.AnyType)
	case map[string]string:
		return cel.MapType(cel.StringType, cel.StringType)
	case []any:
		return cel.ListType(cel.AnyType)
	case []string:
		return cel.ListType(cel.StringType)
	default:
		return cel.AnyType
	}
}

// applyVariableSubstitution applies variable substitution to all text content in the MD
func (md *MD) applyVariableSubstitution() error {
	if md.Frontmatter == nil || md.Frontmatter.Variables == nil {
		return nil
	}

	// Convert Variables map[string]string to map[string]any for expandTemplate
	store := make(map[string]any)
	for k, v := range md.Frontmatter.Variables {
		store[k] = v
	}

	// Apply substitution to each content
	for _, content := range md.Contents {
		// Apply to titles
		for i, title := range content.Titles {
			expanded, err := expandTemplate(title, store)
			if err != nil {
				return fmt.Errorf("failed to expand title template: %w", err)
			}
			content.Titles[i] = expanded
		}

		// Apply to title bodies
		for _, body := range content.TitleBodies {
			if err := applyVariableSubstitutionToBody(body, store); err != nil {
				return fmt.Errorf("failed to expand title body template: %w", err)
			}
		}

		// Apply to subtitles
		for i, subtitle := range content.Subtitles {
			expanded, err := expandTemplate(subtitle, store)
			if err != nil {
				return fmt.Errorf("failed to expand subtitle template: %w", err)
			}
			content.Subtitles[i] = expanded
		}

		// Apply to subtitle bodies
		for _, body := range content.SubtitleBodies {
			if err := applyVariableSubstitutionToBody(body, store); err != nil {
				return fmt.Errorf("failed to expand subtitle body template: %w", err)
			}
		}

		// Apply to bodies
		for _, body := range content.Bodies {
			if err := applyVariableSubstitutionToBody(body, store); err != nil {
				return fmt.Errorf("failed to expand body template: %w", err)
			}
		}

		// Apply to comments (speaker notes)
		for i, comment := range content.Comments {
			expanded, err := expandTemplate(comment, store)
			if err != nil {
				return fmt.Errorf("failed to expand comment template: %w", err)
			}
			content.Comments[i] = expanded
		}

		// Apply to headings
		for level, headings := range content.Headings {
			for i, heading := range headings {
				expanded, err := expandTemplate(heading, store)
				if err != nil {
					return fmt.Errorf("failed to expand heading template: %w", err)
				}
				content.Headings[level][i] = expanded
			}
		}
	}

	return nil
}

// applyVariableSubstitutionToBody applies variable substitution to all fragments in a body
func applyVariableSubstitutionToBody(body *deck.Body, store map[string]any) error {
	if body == nil {
		return nil
	}

	for _, paragraph := range body.Paragraphs {
		for _, fragment := range paragraph.Fragments {
			expanded, err := expandTemplate(fragment.Value, store)
			if err != nil {
				return fmt.Errorf("failed to expand fragment template: %w", err)
			}
			fragment.Value = expanded
		}
	}

	return nil
}
