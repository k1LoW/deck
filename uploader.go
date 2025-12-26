package deck

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/k1LoW/exec"
)

// Environment variable names for external uploader CLI.
const (
	EnvUploadMIME     = "DECK_UPLOAD_MIME"
	EnvUploadFilename = "DECK_UPLOAD_FILENAME"
	EnvDeleteID       = "DECK_DELETE_ID"
)

// externalUploader implements image upload/delete using external CLI commands.
type externalUploader struct {
	uploadCmd string
	deleteCmd string
}

// newExternalUploader creates a new externalUploader.
func newExternalUploader(uploadCmd, deleteCmd string) *externalUploader {
	return &externalUploader{
		uploadCmd: uploadCmd,
		deleteCmd: deleteCmd,
	}
}

// Upload uploads an image using the external upload command.
// It passes image data via stdin and sets environment variables DECK_UPLOAD_MIME and DECK_UPLOAD_FILENAME.
// The command should output the public URL on the first line and resource ID on the second line.
func (u *externalUploader) Upload(ctx context.Context, data []byte, mimeType, filename string) (publicURL, resourceID string, err error) {
	c, args, err := buildCommand(u.uploadCmd)
	if err != nil {
		return "", "", fmt.Errorf("failed to build upload command: %w", err)
	}

	cmd := exec.CommandContext(ctx, c, args...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvUploadMIME, mimeType))
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvUploadFilename, filename))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("failed to run upload command: %w\nstderr: %s", err, stderr.String())
	}

	// Parse stdout: first line is public URL, second line is resource ID
	scanner := bufio.NewScanner(&stdout)
	if !scanner.Scan() {
		return "", "", fmt.Errorf("upload command did not output public URL")
	}
	publicURL = strings.TrimSpace(scanner.Text())

	if !scanner.Scan() {
		return "", "", fmt.Errorf("upload command did not output resource ID")
	}
	resourceID = strings.TrimSpace(scanner.Text())

	if publicURL == "" {
		return "", "", fmt.Errorf("upload command returned empty public URL")
	}
	if resourceID == "" {
		return "", "", fmt.Errorf("upload command returned empty resource ID")
	}

	return publicURL, resourceID, nil
}

// Delete deletes an uploaded image using the external delete command.
// It sets the environment variable DECK_DELETE_ID with the resource ID.
func (u *externalUploader) Delete(ctx context.Context, resourceID string) error {
	if u.deleteCmd == "" {
		// No delete command configured, skip deletion
		return nil
	}

	c, args, err := buildCommand(u.deleteCmd)
	if err != nil {
		return fmt.Errorf("failed to build delete command: %w", err)
	}

	cmd := exec.CommandContext(ctx, c, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", EnvDeleteID, resourceID))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run delete command: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// buildCommand parses a command string and returns the command and arguments.
// This is similar to the implementation in md/md.go.
func buildCommand(cmdStr string) (string, []string, error) {
	shell, err := detectShell()
	if err != nil {
		return "", nil, err
	}
	return shell, []string{"-c", cmdStr}, nil
}

// detectShell detects the current shell.
func detectShell() (string, error) {
	shells := []string{
		os.Getenv("SHELL"),
		"/bin/bash",
		"/bin/sh",
	}
	for _, shell := range shells {
		if shell == "" {
			continue
		}
		if _, err := os.Stat(shell); err == nil {
			return shell, nil
		}
	}
	return "", fmt.Errorf("failed to detect shell")
}

// generateTempFilename generates a temporary filename for uploaded images.
func generateTempFilename() string {
	return fmt.Sprintf("________tmp-for-deck-%s", time.Now().Format(time.RFC3339))
}
