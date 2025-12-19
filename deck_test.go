package deck

import (
	"strings"
	"testing"

	"google.golang.org/api/slides/v1"
)

func TestCountString(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"👉", 2},
		{"➡️", 2},
		{"👍🏼", 4},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := countString(tt.in)
			if got != tt.want {
				t.Errorf("countString(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestWithProfile(t *testing.T) {
	tests := []struct {
		name        string
		profile     string
		expectError bool
	}{
		// Valid profile names
		{
			name:        "valid profile with hyphen",
			profile:     "valid-profile",
			expectError: false,
		},
		{
			name:        "valid profile with underscore and numbers",
			profile:     "profile_123",
			expectError: false,
		},
		{
			name:        "valid profile with mixed case",
			profile:     "ProfileABC",
			expectError: false,
		},
		{
			name:        "valid profile with complex pattern",
			profile:     "test-profile_001",
			expectError: false,
		},
		{
			name:        "empty string should be valid",
			profile:     "",
			expectError: false,
		},
		{
			name:        "only alphanumeric",
			profile:     "abc123XYZ",
			expectError: false,
		},
		{
			name:        "only underscores",
			profile:     "___",
			expectError: false,
		},
		{
			name:        "only hyphens",
			profile:     "---",
			expectError: false,
		},
		{
			name:        "single character valid",
			profile:     "a",
			expectError: false,
		},
		// Invalid profile names
		{
			name:        "profile with space",
			profile:     "profile with space",
			expectError: true,
		},
		{
			name:        "profile with at symbol",
			profile:     "profile@email",
			expectError: true,
		},
		{
			name:        "profile with japanese characters",
			profile:     "プロファイル",
			expectError: true,
		},
		{
			name:        "profile with slash",
			profile:     "profile/slash",
			expectError: true,
		},
		{
			name:        "profile with dot",
			profile:     "profile.dot",
			expectError: true,
		},
		{
			name:        "profile with parentheses",
			profile:     "profile(test)",
			expectError: true,
		},
		{
			name:        "profile with brackets",
			profile:     "profile[test]",
			expectError: true,
		},
		{
			name:        "profile with special characters",
			profile:     "profile!@#$%",
			expectError: true,
		},
		{
			name:        "profile with backslash",
			profile:     "profile\\test",
			expectError: true,
		},
		{
			name:        "profile with colon",
			profile:     "profile:test",
			expectError: true,
		},
		// Edge cases
		{
			name:        "very long valid profile",
			profile:     "a123456789012345678901234567890123456789012345678901234567890_test-profile",
			expectError: false,
		},
		{
			name:        "profile with tab character",
			profile:     "profile\ttest",
			expectError: true,
		},
		{
			name:        "profile with newline",
			profile:     "profile\ntest",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a dummy Deck to test the option
			deck := &Deck{}

			// Apply the WithProfile option
			opt := WithProfile(tt.profile)
			err := opt(deck)

			if tt.expectError {
				if err == nil {
					t.Errorf("WithProfile(%q) expected error but got none", tt.profile)
				}
			} else {
				if err != nil {
					t.Errorf("WithProfile(%q) expected no error but got: %v", tt.profile, err)
				} else {
					// Verify that profile is set correctly for valid cases
					if deck.profile != tt.profile {
						t.Errorf("WithProfile(%q) profile not set correctly: got %q, want %q", tt.profile, deck.profile, tt.profile)
					}
				}
			}
		})
	}
}

func TestValidateLayouts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		slides             Slides
		availableLayouts   []string
		defaultTitleLayout string
		defaultLayout      string
		wantErr            bool
		wantErrContains    []string
	}{
		{
			name:               "empty slides",
			slides:             Slides{},
			availableLayouts:   []string{"Title Slide", "Title and Content"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Title and Content",
			wantErr:            false,
		},
		{
			name: "valid layout",
			slides: Slides{
				{Layout: "Title Slide"},
				{Layout: "Title and Content"},
			},
			availableLayouts:   []string{"Title Slide", "Title and Content", "Section Header"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Title and Content",
			wantErr:            false,
		},
		{
			name: "invalid layout",
			slides: Slides{
				{Layout: "Title Slide"},
				{Layout: "section"}, // typo: should be "Section Header"
			},
			availableLayouts:   []string{"Title Slide", "Title and Content", "Section Header"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Title and Content",
			wantErr:            true,
			wantErrContains:    []string{"section"},
		},
		{
			name: "empty layout uses default",
			slides: Slides{
				{Layout: ""}, // should use defaultTitleLayout
				{Layout: ""}, // should use defaultLayout
			},
			availableLayouts:   []string{"Title Slide", "Title and Content"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Title and Content",
			wantErr:            false,
		},
		{
			name: "multiple invalid layouts",
			slides: Slides{
				{Layout: "invalid1"},
				{Layout: "invalid2"},
			},
			availableLayouts:   []string{"Title Slide", "Title and Content"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Title and Content",
			wantErr:            true,
			wantErrContains:    []string{"invalid1", "invalid2"},
		},
		{
			name: "invalid default title layout",
			slides: Slides{
				{Layout: ""}, // uses defaultTitleLayout which doesn't exist
			},
			availableLayouts:   []string{"Title and Content"},
			defaultTitleLayout: "Non-existent Title Layout",
			defaultLayout:      "Title and Content",
			wantErr:            true,
			wantErrContains:    []string{"Non-existent Title Layout"},
		},
		{
			name: "invalid default layout",
			slides: Slides{
				{Layout: "Title Slide"},
				{Layout: ""}, // uses defaultLayout which doesn't exist
			},
			availableLayouts:   []string{"Title Slide"},
			defaultTitleLayout: "Title Slide",
			defaultLayout:      "Non-existent Layout",
			wantErr:            true,
			wantErrContains:    []string{"Non-existent Layout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a Deck with mock presentation layouts
			layouts := make([]*slides.Page, len(tt.availableLayouts))
			for i, name := range tt.availableLayouts {
				layouts[i] = &slides.Page{
					LayoutProperties: &slides.LayoutProperties{
						DisplayName: name,
					},
				}
			}
			d := &Deck{
				presentation: &slides.Presentation{
					Layouts: layouts,
				},
				defaultTitleLayout: tt.defaultTitleLayout,
				defaultLayout:      tt.defaultLayout,
			}

			err := d.validateLayouts(tt.slides)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateLayouts() expected error but got none")
					return
				}
				for _, want := range tt.wantErrContains {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("validateLayouts() error = %v, want error containing %q", err, want)
					}
				}
			} else {
				if err != nil {
					t.Errorf("validateLayouts() unexpected error: %v", err)
				}
			}
		})
	}
}
