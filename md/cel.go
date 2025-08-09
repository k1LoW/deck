package md

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
)

func (md *MD) reflectDefaults() error {
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
