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
	"os"
	"path/filepath"
	"strings"

	"github.com/Songmu/prompter"
	"github.com/fatih/color"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
	"github.com/k1LoW/errors"
	"github.com/spf13/cobra"
)

var out string

var exportCmd = &cobra.Command{
	Use:   "export [DECK_FILE]",
	Short: "export deck",
	Long:  `export deck.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) > 0 {
			f := args[0]
			markdownData, err := md.ParseFile(f, nil)
			if err != nil {
				// Deprecated
				presentationID = f
				cmd.Println(color.YellowString("WARNING: The argument is deprecated. Use --presentation-id instead."))
			} else if out == "" {
				// If no output destination is specified, output to a file name based on the MD file name.
				// We could also consider generating file names from presentation names or titles, but from a
				// local file management perspective, it is more intuitive to use the same file name.
				// Furthermore, if we use presentation names, we need to sanitize the string to make filesystem safe.
				// So, following the MD file name is a good default.
				out = strings.TrimSuffix(filepath.Base(f), filepath.Ext(f)) + ".pdf"
			}
			if presentationID == "" && markdownData.Frontmatter != nil && markdownData.Frontmatter.PresentationID != "" {
				presentationID = markdownData.Frontmatter.PresentationID
			}
		}
		if presentationID == "" {
			return fmt.Errorf("presentation ID is required. Use --presentation-id or set it in the frontmatter of the markdown file")
		}
		if out == "" {
			// If the presentationID is passed as an argument (not recommended), "deck.pdf" will be used as a default output name.
			out = "deck.pdf"
		}

		opts := []deck.Option{
			deck.WithProfile(profile),
			deck.WithPresentationID(presentationID),
		}
		d, err := deck.New(ctx, opts...)
		if err != nil {
			if errors.Is(err, deck.HTTPClientError) {
				cmd.Println(setupInstructionMessage)
			}
			return err
		}
		if _, err = os.Stat(out); err == nil {
			if !prompter.YN(fmt.Sprintf("%q already exists. Do you want to overwrite it?", out), false) {
				cmd.Println("The export has been canceled.")
				return nil
			}
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		if err := d.Export(ctx, f); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVarP(&presentationID, "presentation-id", "i", "", "Google Slides presentation ID")
	exportCmd.Flags().StringVarP(&out, "out", "o", "", `output file (default: follow the md file name, or "deck.pdf")`)
}
