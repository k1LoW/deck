package deck

import (
	"testing"
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
