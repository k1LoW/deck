package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/config"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/slides/v1"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check deck environment and configuration",
	Long:  `Check deck environment and configuration to ensure everything is set up correctly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Color setup
		green := color.New(color.FgGreen)
		red := color.New(color.FgRed)
		yellow := color.New(color.FgYellow)
		bold := color.New(color.Bold)

		allOK := true

		// 1. Check credentials file
		cmd.Print("🔍 Checking credentials file ... ")

		credPath := deck.GetCredentialsPath(profile)
		if _, err := os.Stat(credPath); os.IsNotExist(err) {
			red.Println("✗ NOT FOUND")
			cmd.Printf("   Expected at: %s\n", credPath)
			allOK = false
		} else {
			// Check JSON format
			credData, err := os.ReadFile(credPath)
			if err != nil {
				red.Println("✗ READ ERROR")
				cmd.Printf("   Error reading file: %v\n", err)
				allOK = false
			} else {
				// Test if JSON can be parsed as OAuth config
				_, err := google.ConfigFromJSON(credData, slides.PresentationsScope, slides.DriveScope)
				if err != nil {
					red.Println("✗ INVALID OAUTH CONFIG")
					cmd.Printf("   OAuth configuration error: %v\n", err)
					allOK = false
				} else {
					green.Println("✓ OK")
					cmd.Printf("   Credentials file: %s\n", credPath)
				}
			}
		}

		if !allOK {
			cmd.Println()
			showSetupHelp(cmd)
			return nil
		}

		// 2. Check login / HTTP client creation
		cmd.Print("🔐 Checking authentication ... ")

		// Try to create deck client (this validates OAuth credentials)
		err := deck.Doctor(ctx, deck.WithProfile(profile))
		if err != nil {
			red.Println("✗ AUTH FAILED")
			cmd.Printf("   Authentication error: %v\n", err)
			allOK = false
		} else {
			green.Println("✓ OK")
			cmd.Println("   Successfully connected to Google API")
		}

		if !allOK {
			cmd.Println()
			showSetupHelp(cmd)
			return nil
		}

		// 3. Check configuration file (optional)
		cmd.Print("🔧 Checking configuration file ... ")

		_, err = config.Load(profile)
		if err != nil {
			yellow.Println("⚠️ CONFIG ERROR")
			cmd.Printf("   Error loading config: %v\n", err)
			allOK = false
		} else {
			green.Println("✓ OK")
			cmd.Println("   Configuration loaded successfully")
		}

		// Final message
		cmd.Println()
		if allOK {
			bold.Printf("🎉 ")
			green.Print("All checks passed! You are ready to use deck")
			bold.Println(".")
			cmd.Println()
			cmd.Println("Try creating a new presentation:")
			yellow.Println("  deck new")
		} else {
			red.Println("⚠️  Setup is incomplete.")
			cmd.Println("\nPlease fix the issues above to use deck properly.")
		}

		return nil
	},
}

func showSetupHelp(cmd *cobra.Command) {
	yellow := color.New(color.FgYellow)
	bold := color.New(color.Bold)
	cyan := color.New(color.FgCyan)

	bold.Println("📚 Setup Guide")
	cmd.Println()
	cmd.Println("To use deck, you need to set up OAuth client credentials.")
	cmd.Println()
	bold.Println("Follow these steps:")
	cmd.Println()
	bold.Print("1. ")
	cmd.Println("Create (or reuse) a developer project at Google Cloud Console")
	cyan.Println("   https://console.cloud.google.com")
	cmd.Println()
	bold.Print("2. ")
	cmd.Println("Enable Google Slides API and Google Drive API")
	cyan.Println("   https://console.cloud.google.com/apis/library/slides.googleapis.com")
	cyan.Println("   https://console.cloud.google.com/apis/library/drive.googleapis.com")
	cmd.Println()
	bold.Print("3. ")
	cmd.Println("Go to Credentials page and create OAuth 2.0 Client ID")
	cyan.Println("   https://console.cloud.google.com/apis/credentials")
	cmd.Println("   - Choose type: Desktop app")
	cmd.Println("   - Add your email address as a test user if needed")
	cmd.Println()
	bold.Print("4. ")
	cmd.Printf("Download and save the credentials JSON file as:\n")
	yellow.Printf("   %s\n", deck.GetCredentialsPath(profile))
	cmd.Println()
	bold.Println("📖 For detailed step-by-step instructions, see:")
	cyan.Println("   https://github.com/k1LoW/deck#get-and-set-your-oauth-client-credentials")
	cmd.Println()
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
