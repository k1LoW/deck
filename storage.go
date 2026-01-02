package deck

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/k1LoW/deck/template"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/exec"
	"google.golang.org/api/drive/v3"
)

// Storage is the interface for image upload/delete operations.
type Storage interface {
	Upload(ctx context.Context, data []byte, mimeType string) (publicURL, resourceID string, err error)
	Delete(ctx context.Context, resourceID string) error
}

// googleDriveStorage implements Storage using Google Drive.
type googleDriveStorage struct {
	driveSrv             *drive.Service
	folderID             string
	allowReadingByAnyone func(ctx context.Context, fileID string) error
	deleteOrTrash        func(ctx context.Context, fileID string) error
}

// newGoogleDriveStorage creates a new googleDriveStorage.
func newGoogleDriveStorage(
	driveSrv *drive.Service,
	folderID string,
	allowReadingByAnyone func(ctx context.Context, fileID string) error,
	deleteOrTrash func(ctx context.Context, fileID string) error,
) *googleDriveStorage {
	return &googleDriveStorage{
		driveSrv:             driveSrv,
		folderID:             folderID,
		allowReadingByAnyone: allowReadingByAnyone,
		deleteOrTrash:        deleteOrTrash,
	}
}

// Upload uploads an image to Google Drive.
func (u *googleDriveStorage) Upload(ctx context.Context, data []byte, mimeType string) (publicURL, resourceID string, err error) {
	df := &drive.File{
		Name:     fmt.Sprintf("________tmp-for-deck-%s", time.Now().Format(time.RFC3339)),
		MimeType: mimeType,
	}
	if u.folderID != "" {
		df.Parents = []string{u.folderID}
	}

	uploaded, err := u.driveSrv.Files.Create(df).Media(bytes.NewBuffer(data)).SupportsAllDrives(true).Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to upload image: %w", err)
	}
	resourceID = uploaded.Id

	defer func() {
		if err != nil {
			// Clean up uploaded file on error
			if deleteErr := u.deleteOrTrash(ctx, uploaded.Id); deleteErr != nil {
				err = errors.Join(err, deleteErr)
			}
		}
	}()

	// To specify a URL for CreateImageRequest, we must make the webContentURL readable to anyone
	// and configure the necessary permissions for this purpose.
	if err = u.allowReadingByAnyone(ctx, uploaded.Id); err != nil {
		return "", "", fmt.Errorf("failed to set permission for image: %w", err)
	}

	// Get webContentLink
	f, err := u.driveSrv.Files.Get(uploaded.Id).Fields("webContentLink").SupportsAllDrives(true).Do()
	if err != nil {
		return "", "", fmt.Errorf("failed to get webContentLink for image: %w", err)
	}

	if f.WebContentLink == "" {
		return "", "", fmt.Errorf("webContentLink is empty for image: %s", uploaded.Id)
	}
	publicURL = f.WebContentLink

	return publicURL, resourceID, nil
}

// Delete deletes an uploaded image from Google Drive.
func (u *googleDriveStorage) Delete(ctx context.Context, resourceID string) error {
	return u.deleteOrTrash(ctx, resourceID)
}

// externalStorage implements Storage using external CLI commands.
type externalStorage struct {
	uploadCmd string
	deleteCmd string
}

// newExternalStorage creates a new externalStorage.
func newExternalStorage(uploadCmd, deleteCmd string) *externalStorage {
	return &externalStorage{
		uploadCmd: uploadCmd,
		deleteCmd: deleteCmd,
	}
}

// Upload uploads an image using the external upload command.
// It passes image data via stdin and sets the environment variable DECK_UPLOAD_MIME.
// The command also supports template variables: {{mime}} and {{env.XXX}}.
// The command should output the public URL on the first line and resource ID on the second line.
func (u *externalStorage) Upload(ctx context.Context, data []byte, mimeType string) (publicURL, resourceID string, err error) {
	const envUploadMIME = "DECK_UPLOAD_MIME"

	// Prepare environment variables
	env := template.EnvironToMap()
	env[envUploadMIME] = mimeType

	// Prepare template store
	store := map[string]any{
		"mime": mimeType,
		"env":  env,
	}

	// Expand template in command
	expandedCmd, err := template.Expand(u.uploadCmd, store)
	if err != nil {
		return "", "", fmt.Errorf("failed to expand upload command template: %w", err)
	}

	c, args, err := buildCommand(expandedCmd)
	if err != nil {
		return "", "", fmt.Errorf("failed to build upload command: %w", err)
	}

	cmd := exec.CommandContext(ctx, c, args...)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, envUploadMIME+"="+mimeType)

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
// The command also supports template variables: {{id}} and {{env.XXX}}.
func (u *externalStorage) Delete(ctx context.Context, resourceID string) error {
	const envDeleteID = "DECK_DELETE_ID"

	if u.deleteCmd == "" {
		// No delete command configured, skip deletion
		return nil
	}

	// Prepare environment variables
	env := template.EnvironToMap()
	env[envDeleteID] = resourceID

	// Prepare template store
	store := map[string]any{
		"id":  resourceID,
		"env": env,
	}

	// Expand template in command
	expandedCmd, err := template.Expand(u.deleteCmd, store)
	if err != nil {
		return fmt.Errorf("failed to expand delete command template: %w", err)
	}

	c, args, err := buildCommand(expandedCmd)
	if err != nil {
		return fmt.Errorf("failed to build delete command: %w", err)
	}

	cmd := exec.CommandContext(ctx, c, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, envDeleteID+"="+resourceID)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run delete command: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

// buildCommand parses a command string and returns the command and arguments.
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
