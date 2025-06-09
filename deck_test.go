package deck

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func TestApplyAction(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		// t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}
	ctx := context.Background()
	//presentationID := os.Getenv("TEST_PRESENTATION_ID")
	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	var tests = []struct {
		name     string
		before   Slides
		actions  func(t *testing.T, d *Deck)
		expected Slides
	}{
		{
			name: "append slide (index 1)",
			before: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
			actions: func(t *testing.T, d *Deck) {
				if err := d.appendPage(ctx, &Slide{Layout: "title-and-body", Titles: []string{"Slide 2"}}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
			},
		},
		{
			name:   "append slide (index 0)",
			before: Slides{},
			actions: func(t *testing.T, d *Deck) {
				if err := d.appendPage(ctx, &Slide{Layout: "title-and-body", Titles: []string{"Slide 1"}}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
		},
		{
			name: "insert slide (index 0)",
			before: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
			actions: func(t *testing.T, d *Deck) {
				if err := d.insertPage(ctx, 0, &Slide{Layout: "title-and-body", Titles: []string{"Slide 2"}}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
		},
		{
			name: "insert slide (index 1)",
			before: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
			},
			actions: func(t *testing.T, d *Deck) {
				if err := d.insertPage(ctx, 1, &Slide{Layout: "title-and-body", Titles: []string{"Slide 1.5"}}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
				{Layout: "title-and-body", Titles: []string{"Slide 1.5"}},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
			},
		},
		{
			name: "move slide (index 1)",
			before: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
			},
			actions: func(t *testing.T, d *Deck) {
				if err := d.movePage(ctx, 1, 0); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
		},
		{
			name: "move slide (index 0)",
			before: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
			},
			actions: func(t *testing.T, d *Deck) {
				if err := d.movePage(ctx, 0, 1); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}},
			},
		},
		{
			name: "diffSlides generated move operations",
			before: Slides{
				{Layout: "title", Titles: []string{"Delete Me 1"}},
				{Layout: "title", Titles: []string{"Delete Me 2"}},
				{Layout: "title", Titles: []string{"Keep Me A"}},
				{Layout: "title", Titles: []string{"Keep Me B"}},
			},
			actions: func(t *testing.T, d *Deck) {
				// Test using the actual diffSlides generated actions
				targetSlides := Slides{
					{Layout: "title", Titles: []string{"Keep Me B"}},
					{Layout: "title", Titles: []string{"Keep Me A"}},
					{Layout: "title", Titles: []string{"New Page"}},
				}

				// Apply the target slides using diffSlides
				if err := d.Apply(ctx, targetSlides); err != nil {
					t.Fatalf("failed to apply target slides: %v", err)
				}
			},
			expected: Slides{
				{Layout: "title", Titles: []string{"Keep Me B"}},
				{Layout: "title", Titles: []string{"Keep Me A"}},
				{Layout: "title", Titles: []string{"New Page"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(ctx, WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.DeletePageAfter(ctx, 0); err != nil {
				t.Fatal(err)
			}
			if err := d.DeletePage(ctx, 0); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, tt.before); err != nil {
				t.Fatal(err)
			}
			before, err := d.DumpSlides(ctx)
			if diff := cmp.Diff(tt.before, before, cmpopts...); diff != "" {
				t.Errorf("before apply: %s", diff)
			}
			if err := d.refresh(ctx); err != nil {
				t.Fatal(err)
			}
			t.Log("---")

			tt.actions(t, d)

			actual, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.expected, actual, cmpopts...); diff != "" {
				t.Error(diff)
			}
		})
	}
}
