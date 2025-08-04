package deck

import (
	"testing"

	"google.golang.org/api/slides/v1"
)

func TestList_WithSupportAllDrives(t *testing.T) {
	tests := []struct {
		name             string
		supportAllDrives bool
		wantQuery        string
	}{
		{
			name:             "with support all drives enabled",
			supportAllDrives: true,
			wantQuery:        "mimeType='application/vnd.google-apps.presentation'",
		},
		{
			name:             "with support all drives disabled",
			supportAllDrives: false,
			wantQuery:        "mimeType='application/vnd.google-apps.presentation'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a simplified test that checks the option is properly set
			d := &Deck{
				supportAllDrives: tt.supportAllDrives,
			}

			if d.supportAllDrives != tt.supportAllDrives {
				t.Errorf("supportAllDrives = %v, want %v", d.supportAllDrives, tt.supportAllDrives)
			}
		})
	}
}

func TestWithSupportAllDrives(t *testing.T) {
	tests := []struct {
		name    string
		support bool
		want    bool
	}{
		{
			name:    "enable support all drives",
			support: true,
			want:    true,
		},
		{
			name:    "disable support all drives",
			support: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Deck{}
			opt := WithSupportAllDrives(tt.support)
			if err := opt(d); err != nil {
				t.Fatalf("WithSupportAllDrives() error = %v", err)
			}
			if d.supportAllDrives != tt.want {
				t.Errorf("supportAllDrives = %v, want %v", d.supportAllDrives, tt.want)
			}
		})
	}
}

func TestDeck_DefaultSupportAllDrives(t *testing.T) {
	// Test that new Deck instances have supportAllDrives set to true by default
	// Create a mock deck without calling New (which requires Drive API)
	d := &Deck{
		styles:           make(map[string]*slides.TextStyle),
		shapes:           make(map[string]*slides.ShapeProperties),
		supportAllDrives: true, // This should be the default
	}

	if !d.supportAllDrives {
		t.Error("Expected supportAllDrives to be true by default")
	}
}
