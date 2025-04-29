/*
Copyright © 2025 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/handler/dot"
	"github.com/k1LoW/deck/md"
	"github.com/spf13/cobra"
)

var (
	title   string
	page    string
	watch   bool
	verbose bool
	logger  *slog.Logger
)

var applyCmd = &cobra.Command{
	Use:   "apply [PRESENTATION_ID] [DECK_FILE]",
	Short: "apply desk written in markdown to Google Slides presentation",
	Long:  `apply desk written in markdown to Google Slides presentation.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if page != "" && watch {
			return fmt.Errorf("cannot use --page and --watch together")
		}
		return cobra.ExactArgs(2)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		id := args[0]
		f := args[1]
		contents, err := md.ParseFile(f)
		if err != nil {
			return err
		}
		if verbose {
			logger = slog.New(
				slog.NewTextHandler(os.Stdout, nil),
			)
		} else {
			logger = slog.New(
				dot.New(slog.NewTextHandler(os.Stdout, nil)),
			)
		}
		opts := []deck.Option{
			deck.WithPresentationID(id),
			deck.WithLogger(logger),
		}
		d, err := deck.New(ctx, opts...)
		if err != nil {
			return err
		}
		if title != "" {
			if err := d.UpdateTitle(ctx, title); err != nil {
				return err
			}
		}
		if watch {
			if err := d.Apply(ctx, contents.ToSlides()); err != nil {
				return err
			}
			logger.Info("initial apply completed", slog.String("presentation_id", id))

			return watchFile(cmd.Context(), f, contents, d)
		} else {
			pages, err := pageToPages(page, len(contents))
			if err != nil {
				return err
			}
			if err := d.ApplyPages(ctx, contents.ToSlides(), pages); err != nil {
				return err
			}
			logger.Info("apply completed", slog.String("presentation_id", id))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&title, "title", "t", "", "title of the presentation")
	applyCmd.Flags().StringVarP(&page, "page", "p", "", "page to apply")
	applyCmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch for changes")
	applyCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

func pageToPages(page string, total int) ([]int, error) {
	if page == "" {
		// If no page is specified, return all pages
		pages := make([]int, total)
		for i := 0; i < total; i++ {
			pages[i] = i + 1
		}
		return pages, nil
	}

	var result []int
	// Split by comma to handle comma-separated list
	parts := strings.Split(page, ",")

	for _, part := range parts {
		// Check if it's a range (contains "-")
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")

			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}

			start, end := rangeParts[0], rangeParts[1]

			var startPage, endPage int
			var err error

			if start == "" {
				// Open start range: "-5"
				startPage = 1
			} else {
				// Parse start page
				startPage, err = strconv.Atoi(start)
				if err != nil {
					return nil, fmt.Errorf("invalid page number: %s", start)
				}
			}

			if end == "" {
				// Open end range: "3-"
				endPage = total
			} else {
				// Parse end page
				endPage, err = strconv.Atoi(end)
				if err != nil {
					return nil, fmt.Errorf("invalid page number: %s", end)
				}
			}

			// Validate page range
			if startPage < 1 || startPage > total || endPage < 1 || endPage > total || startPage > endPage {
				return nil, fmt.Errorf("invalid page range: %s (total pages: %d)", part, total)
			}

			// Add all pages in the range
			for i := startPage; i <= endPage; i++ {
				result = append(result, i)
			}
		} else {
			// Single page number
			pageNum, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number: %s", part)
			}

			// Validate page number
			if pageNum < 1 || pageNum > total {
				return nil, fmt.Errorf("page number out of range: %d (total pages: %d)", pageNum, total)
			}

			result = append(result, pageNum)
		}
	}

	return result, nil
}

// watchFile watches for changes in the file and applies them to the presentation.
func watchFile(ctx context.Context, filePath string, oldContents md.Contents, d *deck.Deck) error {
	// Get the absolute path of the file
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(absPath)
	fileName := filepath.Base(absPath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := watcher.Add(dir); err != nil {
		return err
	}

	logger.Info("watching for changes", slog.String("file", absPath))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}

			if filepath.Base(event.Name) != fileName {
				continue
			}

			if event.Op&fsnotify.Write != fsnotify.Write {
				continue
			}

			logger.Info("file modified", slog.String("file", event.Name))

			var newContents md.Contents
			var parseErr error

			for retry := 0; retry < 3; retry++ {
				newContents, parseErr = md.ParseFile(filePath)
				if parseErr == nil {
					break
				}

				logger.Warn("failed to parse file, retrying...",
					slog.String("error", parseErr.Error()),
					slog.Int("retry", retry+1))

				time.Sleep(100 * time.Millisecond)
			}

			if parseErr != nil {
				logger.Error("failed to parse file after retries", slog.String("error", parseErr.Error()))
				continue
			}

			changedPages := md.DiffContents(oldContents, newContents)

			if len(changedPages) == 0 {
				logger.Info("no changes detected")
				continue
			}

			logger.Info("detected changes", slog.Any("pages", changedPages))

			if err := d.ApplyPages(ctx, newContents.ToSlides(), changedPages); err != nil {
				logger.Error("failed to apply changes", slog.String("error", err.Error()))
				continue
			}

			logger.Info("applied changes", slog.Any("pages", changedPages))

			oldContents = newContents

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Error("watcher error", slog.String("error", err.Error()))

		case <-ctx.Done():
			return nil
		}
	}
}
