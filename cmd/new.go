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

	"github.com/fatih/color"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/deck/md"
	"github.com/k1LoW/errors"
	"github.com/spf13/cobra"
)

var (
	base     string
	from     string
	folderID string
)

var newCmd = &cobra.Command{
	Use:   "new [markdown-file]",
	Short: "create new presentation",
	Long: `create new presentation.

If a markdown file is specified, frontmatter with title and presentationID will be added to the file.
If the file doesn't exist, it will be created.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Load config to get default folderID if not specified via flag
		cfg, err := config.Load(profile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		basePresentationID := cfg.BasePresentationID
		if from != "" {
			cmd.Println(color.YellowString("WARNING: --from is deprecated. Please use --base flag instead."))
			basePresentationID = from
		}
		if base != "" {
			basePresentationID = base
		}
		if folderID == "" {
			folderID = cfg.FolderID
		}

		opts := []deck.Option{
			deck.WithProfile(profile),
		}
		if folderID != "" {
			opts = append(opts, deck.WithFolderID(folderID))
		}
		d, err := func() (*deck.Deck, error) {
			if basePresentationID != "" {
				return deck.CreateFrom(ctx, basePresentationID, opts...)
			}
			return deck.Create(ctx, opts...)
		}()
		if err != nil {
			if errors.Is(err, deck.HttpClientError) {
				cmd.Println(setupInstructionMessage)
			}
			return err
		}

		if title != "" {
			if err := d.UpdateTitle(ctx, title); err != nil {
				return err
			}
		}

		presentationID := d.ID()

		// If markdown file is specified, apply frontmatter to it.
		if len(args) > 0 {
			mdFile := args[0]
			if err := md.ApplyFrontmatterToMD(mdFile, title, presentationID); err != nil {
				return err
			}
			cmd.PrintErrf("Applied frontmatter to %s\n", mdFile)
		}
		fmt.Println(presentationID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVarP(&title, "title", "t", "", "title of the presentation")
	newCmd.Flags().StringVarP(&from, "base", "b", "", "base presentation id that uses the theme you want to use")
	newCmd.Flags().StringVarP(&from, "from", "f", "", "(DEPRECATED, use --base/-b) presentation id that uses the theme you want to use")
	newCmd.Flags().StringVarP(&folderID, "folder-id", "", "", "folder id to create the presentation in")
}
