package md

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
)

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
