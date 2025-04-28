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
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/handler/dot"
	"github.com/k1LoW/deck/md"
	"github.com/spf13/cobra"
)

var (
	title string
	page  string
)

var applyCmd = &cobra.Command{
	Use:   "apply [PRESENTATION_ID] [DECK_FILE]",
	Short: "apply desk written in markdown to Google Slides presentation",
	Long:  `apply desk written in markdown to Google Slides presentation.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		f := args[1]
		contents, err := md.ParseFile(f)
		if err != nil {
			return err
		}
		logger := slog.New(
			dot.New(slog.NewTextHandler(os.Stdout, nil)),
		)
		opts := []deck.Option{
			deck.WithPresentationID(id),
			deck.WithLogger(logger),
		}
		d, err := deck.New(cmd.Context(), opts...)
		if err != nil {
			return err
		}
		pages, err := pageToPages(page, len(contents))
		if err != nil {
			return err
		}
		if err := d.ApplyPages(contents, pages); err != nil {
			return err
		}

		fmt.Println()
		if title != "" {
			if err := d.UpdateTitle(title); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().StringVarP(&title, "title", "t", "", "title of the presentation")
	applyCmd.Flags().StringVarP(&page, "page", "p", "", "page to apply")
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
