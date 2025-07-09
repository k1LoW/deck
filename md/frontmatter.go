package md

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/errors"
)

// ApplyFrontmatterToMD updates or creates a markdown file with frontmatter
func ApplyFrontmatterToMD(mdFile, title, presentationID string) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	// Read existing content or create empty content
	var content []byte
	if c, err := os.ReadFile(mdFile); err == nil {
		content = c
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read file: %w", err)
	}

	const fmSep = "---\n"

	// Check if content starts with frontmatter
	var frontmatter = make(map[string]any)
	var bodyContent = content

	if bytes.HasPrefix(content, []byte(fmSep)) {
		// May have frontmatter, parse it
		stuffs := bytes.SplitN(content, []byte(fmSep), 3)
		if len(stuffs) == 3 {
			maybeFrontmatter := stuffs[1]
			maybeBody := stuffs[2]
			var fm = make(map[string]any)
			if err := yaml.Unmarshal(maybeFrontmatter, &fm); err == nil {
				frontmatter = fm
				bodyContent = maybeBody
			}
		}
	}

	// Update frontmatter fields
	if title != "" {
		frontmatter["title"] = title
	}
	frontmatter["presentationId"] = presentationID

	// Marshal frontmatter back to YAML
	frontmatterYAML, err := yaml.Marshal(frontmatter)
	if err != nil {
		return fmt.Errorf("failed to encode frontmatter: %w", err)
	}
	// Remove trailing newline from YAML output if present
	frontmatterYAML = bytes.TrimSpace(frontmatterYAML)

	// Combine frontmatter and body
	var newContent bytes.Buffer
	newContent.WriteString(fmSep)
	newContent.Write(frontmatterYAML)
	newContent.WriteString("\n")
	if !bytes.HasPrefix(bodyContent, []byte(fmSep)) {
		newContent.WriteString(fmSep)
	}
	newContent.Write(bodyContent)

	dir := filepath.Dir(mdFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	// Write the file
	if err := os.WriteFile(mdFile, newContent.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
