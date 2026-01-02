package md

import (
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
)

func (md *MD) reflectDefaults() error {
	if md.Frontmatter == nil {
		return nil
	}
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
		return fmt.Errorf("failed to create environment: %w", err)
	}
	pageTotal := len(md.Contents)
	for i, content := range md.Contents {
		for _, cond := range md.Frontmatter.Defaults {
			ast, issues := env.Compile(fmt.Sprintf("!!(%s)", cond.If))
			if issues != nil && issues.Err() != nil {
				return fmt.Errorf("failed to compile expression: %w", issues.Err())
			}
			prg, err := env.Program(ast)
			if err != nil {
				return fmt.Errorf("failed to create program: %w", err)
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
				return fmt.Errorf("failed to evaluate values: %w", err)
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
	return nil
}

