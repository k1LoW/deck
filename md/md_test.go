package md

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tenntenn/golden"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in string
	}{
		{"../testdata/slide.md"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			slides, err := Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			got, err := json.MarshalIndent(slides, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("UPDATE_GOLDEN") != "" {
				golden.Update(t, "", tt.in, got)
				return
			}
			if diff := golden.Diff(t, "", tt.in, got); diff != "" {
				t.Error(diff)
			}
		})
	}
}
