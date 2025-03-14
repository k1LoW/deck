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
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [PRESENTATION_ID] [DECK_FILE]",
	Short: "apply desk written in markdown to Google Slides presentation",
	Long:  `apply desk written in markdown to Google Slides presentation.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		f := args[1]
		slides, err := md.ParseFile(f)
		if err != nil {
			return err
		}
		d, err := deck.New(cmd.Context(), id)
		if err != nil {
			return err
		}
		if err := d.Apply(slides); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
}
