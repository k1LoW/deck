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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/deck/logger/dot"
	"github.com/k1LoW/deck/md"
	"github.com/k1LoW/errors"
	"github.com/k1LoW/tail"
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/cobra"
)

var (
	presentationID      string
	title               string
	page                string
	watch               bool
	verbosity           int // 1: info, >=2: debug
	logger              *slog.Logger
	codeBlockToImageCmd string
	tb                  = tail.New(30)
)

var applyCmd = &cobra.Command{
	Use:   "apply DECK_FILE",
	Short: "apply desk written in markdown to Google Slides presentation",
	Long:  `apply desk written in markdown to Google Slides presentation.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if page != "" && watch {
			return fmt.Errorf("cannot use --page and --watch together")
		}
		if len(args) == 2 && presentationID != "" {
			return fmt.Errorf("cannot use --presentation-id with two arguments")
		}
		return cobra.RangeArgs(1, 2)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		var f string

		if len(args) == 1 {
			// Single argument case: use frontmatter presentationID
			f = args[0]
		} else {
			// Deprecated: Two argument case: traditional usage
			cmd.Println(color.YellowString("WARNING: Two arguments are deprecated. Please use --presentation-id flag instead."))
			presentationID = args[0]
			f = args[1]
		}
		cfg, err := config.Load(profile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		m, err := md.ParseFile(f, cfg)
		if err != nil {
			return err
		}
		if len(args) == 1 {
			if m.Frontmatter != nil {
				if presentationID == "" && m.Frontmatter.PresentationID != "" {
					presentationID = m.Frontmatter.PresentationID
				}
				if title == "" && m.Frontmatter.Title != "" {
					title = m.Frontmatter.Title
				}
			}
		}

		if presentationID == "" {
			return fmt.Errorf("presentation ID is required, please specify it with --presentation-id or in the frontmatter of the markdown file")
		}

		contents := m.Contents
		tailHandler := slog.NewJSONHandler(tb, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
		if verbosity > 0 {
			var opt *slog.HandlerOptions
			if verbosity > 1 {
				opt = &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}
			}
			logger = slog.New(
				slogmulti.Fanout(
					slog.NewJSONHandler(os.Stdout, opt),
					tailHandler,
				),
			)
		} else {
			h, err := dot.New(slog.NewTextHandler(os.Stdout, nil))
			if err != nil {
				return fmt.Errorf("failed to create dot handler: %w", err)
			}
			logger = slog.New(
				slogmulti.Fanout(
					h,
					tailHandler,
				),
			)
		}
		opts := []deck.Option{
			deck.WithProfile(profile),
			deck.WithPresentationID(presentationID),
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
			slides, err := m.ToSlides(ctx, codeBlockToImageCmd)
			if err != nil {
				return fmt.Errorf("failed to convert markdown contents to slides: %w", err)
			}
			if err := d.Apply(ctx, slides); err != nil {
				return err
			}
			logger.Info("initial apply completed", slog.String("presentation_id", presentationID))

			return watchFile(cmd.Context(), cfg, f, contents, d)
		} else {
			pages, err := pageToPages(page, len(contents))
			if err != nil {
				return err
			}
			slides, err := m.ToSlides(ctx, codeBlockToImageCmd)
			if err != nil {
				return fmt.Errorf("failed to convert markdown contents to slides: %w", err)
			}
			if err := d.ApplyPages(ctx, slides, pages); err != nil {
				return err
			}
			logger.Info("apply completed", slog.String("presentation_id", presentationID), slog.Any("pages", pages))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&presentationID, "presentation-id", "i", "", "Google Slides presentation ID")
	applyCmd.Flags().StringVarP(&title, "title", "t", "", "title of the presentation")
	applyCmd.Flags().StringVarP(&page, "page", "p", "", "page to apply")
	applyCmd.Flags().StringVarP(&codeBlockToImageCmd, "code-block-to-image-command", "c", "", "command to convert code blocks to images")
	applyCmd.Flags().BoolVarP(&watch, "watch", "w", false, "watch for changes")
	applyCmd.Flags().CountVarP(&verbosity, "verbose", "v", "verbose output (can be used multiple times for more verbosity)")
}

func pageToPages(page string, total int) ([]int, error) {
	if page == "" {
		// If no page is specified, return all pages
		pages := make([]int, total)
		for i := range total {
			pages[i] = i + 1
		}
		return pages, nil
	}

	var result []int
	// Split by comma to handle comma-separated list
	for part := range strings.SplitSeq(page, ",") {
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
func watchFile(ctx context.Context, cfg *config.Config, filePath string, oldContents md.Contents, d *deck.Deck) error {
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

	var events []fsnotify.Event
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// drain all stacked events
			events = append(events, event)

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			logger.Error("watcher error", slog.String("error", err.Error()))

		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			fileModified := slices.ContainsFunc(events, func(event fsnotify.Event) bool {
				return filepath.Base(event.Name) == fileName &&
					(event.Op&fsnotify.Write == fsnotify.Write ||
						event.Op&fsnotify.Create == fsnotify.Create)
			})
			events = nil
			if !fileModified {
				continue
			}
			logger.Info("file modified", slog.String("file", fileName))

			newMD, err := md.ParseFile(filePath, cfg)
			if err != nil {
				logger.Error("failed to parse file", slog.String("error", err.Error()))
				continue
			}
			changedPages := md.DiffContents(oldContents, newMD.Contents)

			if len(changedPages) == 0 {
				logger.Info("no changes detected")
				continue
			}

			logger.Info("detected changes", slog.Any("pages", changedPages))
			slides, err := newMD.ToSlides(ctx, codeBlockToImageCmd)
			if err != nil {
				logger.Error("failed to convert markdown contents to slides", slog.String("error", err.Error()))
				continue
			}
			if err := d.ApplyPages(ctx, slides, changedPages); err != nil {
				slogArgs := []any{slog.String("error", err.Error())}
				if verbosity > 1 {
					slogArgs = append(slogArgs, slog.String("stacktrace", errors.StackTraces(err).String()))
				}
				logger.Error("failed to apply changes", slogArgs...)
				continue
			}

			logger.Info("applied changes", slog.Any("pages", changedPages))

			oldContents = newMD.Contents
		}
	}
}
