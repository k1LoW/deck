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

	"github.com/fatih/color"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
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
			markdownData, err := md.ParseFile(f)
			if err != nil {
				// Deprecated
				presentationID = f
				cmd.Println(color.YellowString("WARNING: The argument is deprecated. Use --presentation-id instead."))
				// return err
			}
			if presentationID == "" && markdownData.Frontmatter != nil && markdownData.Frontmatter.PresentationID != "" {
				presentationID = markdownData.Frontmatter.PresentationID
			}
		}
		if presentationID == "" {
			return fmt.Errorf("presentation ID is required. Use --presentation-id or set it in the frontmatter of the markdown file.")
		}

		d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
		if err != nil {
			return err
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
	exportCmd.Flags().StringVarP(&out, "out", "o", "deck.pdf", "output file")
}
