package deck

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestAction(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}
	ctx := context.Background()

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "StyleName"),
		cmpopts.IgnoreFields(Slide{}, "TitleBodies", "SubtitleBodies"),
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
			before: Slides{{
				Layout:      "title-and-body",
				Titles:      []string{"Slide 1"},
				TitleBodies: toBodies([]string{"Slide 1"}),
				Page:        1,
			}},
			actions: func(t *testing.T, d *Deck) {
				if err := d.AppendPage(ctx, &Slide{
					Layout:      "title-and-body",
					Titles:      []string{"Slide 2"},
					TitleBodies: toBodies([]string{"Slide 2"}),
				}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 1},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}, Page: 2},
			},
		},
		{
			name:   "append slide (index 0)",
			before: Slides{},
			actions: func(t *testing.T, d *Deck) {
				if err := d.AppendPage(ctx, &Slide{Layout: "title-and-body",
					Titles: []string{"Slide 1"}, TitleBodies: toBodies([]string{"Slide 1"})}); err != nil {

					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 1},
			},
		},
		{
			name: "insert slide (index 0)",
			before: Slides{{
				Layout:      "title-and-body",
				Titles:      []string{"Slide 1"},
				TitleBodies: toBodies([]string{"Slide 1"}),
				Page:        1,
			}},
			actions: func(t *testing.T, d *Deck) {
				if err := d.InsertPage(ctx, 0, &Slide{
					Layout: "title-and-body", Titles: []string{"Slide 2"}, TitleBodies: toBodies([]string{"Slide 2"})}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}, Page: 1},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 2},
			},
		},
		{
			name: "insert slide (index 1)",
			before: Slides{{
				Layout:      "title-and-body",
				Titles:      []string{"Slide 1"},
				TitleBodies: toBodies([]string{"Slide 1"}),
				Page:        1,
			}, {
				Layout:      "title-and-body",
				Titles:      []string{"Slide 2"},
				TitleBodies: toBodies([]string{"Slide 2"}),
				Page:        2,
			}},
			actions: func(t *testing.T, d *Deck) {
				if err := d.InsertPage(ctx, 1, &Slide{
					Layout: "title-and-body", Titles: []string{"Slide 1.5"}, TitleBodies: toBodies([]string{"Slide 1.5"})}); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 1},
				{Layout: "title-and-body", Titles: []string{"Slide 1.5"}, Page: 2},
				{Layout: "title-and-body", Titles: []string{"Slide 2"}, Page: 3},
			},
		},
		{
			name: "move slide (index 1)",
			before: Slides{{
				Layout:      "title-and-body",
				Titles:      []string{"Slide 1"},
				TitleBodies: toBodies([]string{"Slide 1"}),
				Page:        1,
			}, {
				Layout:      "title-and-body",
				Titles:      []string{"Slide 2"},
				TitleBodies: toBodies([]string{"Slide 2"}),
				Page:        2,
			}},
			actions: func(t *testing.T, d *Deck) {
				if err := d.movePage(ctx, 1, 0); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}, Page: 1},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 2},
			},
		},
		{
			name: "move slide (index 0)",
			before: Slides{{
				Layout:      "title-and-body",
				Titles:      []string{"Slide 1"},
				TitleBodies: toBodies([]string{"Slide 1"}),
				Page:        1,
			}, {
				Layout:      "title-and-body",
				Titles:      []string{"Slide 2"},
				TitleBodies: toBodies([]string{"Slide 2"}),
				Page:        2,
			}},
			actions: func(t *testing.T, d *Deck) {
				if err := d.movePage(ctx, 0, 1); err != nil {
					t.Fatal(err)
				}
			},
			expected: Slides{
				{Layout: "title-and-body", Titles: []string{"Slide 2"}, Page: 1},
				{Layout: "title-and-body", Titles: []string{"Slide 1"}, Page: 2},
			},
		},
		{
			name: "generateActions generated move operations",
			before: Slides{
				{Layout: "title", Titles: []string{"Delete Me 1"}, TitleBodies: toBodies([]string{"Delete Me 1"}), Page: 1},
				{Layout: "title", Titles: []string{"Delete Me 2"}, TitleBodies: toBodies([]string{"Delete Me 2"}), Page: 2},
				{Layout: "title", Titles: []string{"Keep Me A"}, TitleBodies: toBodies([]string{"Keep Me A"}), Page: 3},
				{Layout: "title", Titles: []string{"Keep Me B"}, TitleBodies: toBodies([]string{"Keep Me B"}), Page: 4},
			},
			actions: func(t *testing.T, d *Deck) {
				// Test using the actual generateActions generated actions
				targetSlides := Slides{
					{Layout: "title", Titles: []string{"Keep Me B"}, TitleBodies: toBodies([]string{"Keep Me B"}), Page: 1},
					{Layout: "title", Titles: []string{"Keep Me A"}, TitleBodies: toBodies([]string{"Keep Me A"}), Page: 2},
					{Layout: "title", Titles: []string{"New Page"}, TitleBodies: toBodies([]string{"New Page"}), Page: 3},
				}

				// Apply the target slides using generateActions
				if err := d.Apply(ctx, targetSlides); err != nil {
					t.Fatalf("failed to apply target slides: %v", err)
				}
			},
			expected: Slides{
				{Layout: "title", Titles: []string{"Keep Me B"}, Page: 1},
				{Layout: "title", Titles: []string{"Keep Me A"}, Page: 2},
				{Layout: "title", Titles: []string{"New Page"}, Page: 3},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Acquire a presentation from the pool
			presentationID := AcquirePresentation(t)

			d, err := New(ctx, WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.DeletePageAfter(ctx, 0); err != nil {
				t.Fatal(err)
			}
			if err := d.DeletePages(ctx, []int{0}); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, tt.before); err != nil {
				t.Fatal(err)
			}
			before, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.before, before, cmpopts...); diff != "" {
				t.Errorf("before apply: %s", diff)
			}
			if err := d.refresh(ctx); err != nil {
				t.Fatal(err)
			}
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
