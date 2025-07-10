package deck

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestApply(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	presentationID := os.Getenv("TEST_PRESENTATION_ID")

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "StyleName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(ctx, WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, tt.before); err != nil {
				t.Fatal(err)
			}
			before, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(before, tt.before, cmpopts...); diff != "" {
				t.Error(diff)
			}

			if err := d.Apply(ctx, tt.after); err != nil {
				t.Fatal(err)
			}
			after, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(after, tt.after, cmpopts...); diff != "" {
				t.Error(diff)
			}
		})
	}
}

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
