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

	ctx := context.Background()

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "StyleName"),
		cmpopts.IgnoreFields(Slide{}, "TitleBodies", "SubtitleBodies"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Acquire a presentation from the pool
			presentationID := AcquirePresentation(t)
			defer ReleasePresentation(presentationID)

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
				t.Fatal(diff)
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
