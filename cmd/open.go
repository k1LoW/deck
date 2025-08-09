package cmd

import (
	"fmt"

	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var openPresentationID string

var openCmd = &cobra.Command{
	Use:   "open [DECK_FILE]",
	Short: "open Google Slides presentation in browser",
	Long:  `open Google Slides presentation in browser.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var presentationID string

		if len(args) == 1 {
			f := args[0]
			markdownData, err := md.ParseFile(f, nil)
			if err != nil {
				return err
			}
			if markdownData.Frontmatter != nil && markdownData.Frontmatter.PresentationID != "" {
				presentationID = markdownData.Frontmatter.PresentationID
			}
		}
		// Command line flag takes precedence
		if openPresentationID != "" {
			presentationID = openPresentationID
		}

		if presentationID == "" {
			return fmt.Errorf("presentation ID is required. Use --presentation-id or set it in the frontmatter of the markdown file")
		}

		url := deck.PresentationIDtoURL(presentationID)
		cmd.Println(url)
		return browser.OpenURL(url)
	},
}

func init() {
	rootCmd.AddCommand(openCmd)
	openCmd.Flags().StringVarP(&openPresentationID, "presentation-id", "p", "", "Google Slides presentation ID")
}
