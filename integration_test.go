//go:build integration

package deck_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
)

func TestApply(t *testing.T) {
	presentationID := os.Getenv("TEST_PRESENTATION_ID")

	tests := []struct {
		in string
	}{
		{"testdata/slide.md"},
		{"testdata/cap.md"},
		{"testdata/br.md"},
		{"testdata/list_and_paragraph.md"},
		{"testdata/paragraph_and_list.md"},
		{"testdata/bold_and_italic.md"},
		{"testdata/emoji.md"},
		{"testdata/code.md"},
		{"testdata/style.md"},
		{"testdata/empty_list.md"},
		{"testdata/empty_link.md"},
		{"testdata/single_list.md"},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			contents, err := md.Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			fromMd := contents.ToSlides()
			d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, fromMd); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConvertToSlide(t *testing.T) {
	presentationID := os.Getenv("TEST_PRESENTATION_ID")

	tests := []struct {
		in string
	}{
		{"testdata/slide.md"},
		{"testdata/cap.md"},
		{"testdata/br.md"},
		{"testdata/bold_and_italic.md"},
		// {"testdata/list_and_paragraph.md"},
		//{"testdata/paragraph_and_list.md"},
		{"testdata/emoji.md"},
		{"testdata/code.md"},
		{"testdata/style.md"},
		{"testdata/empty_list.md"},
		{"testdata/empty_link.md"},
	}

	ctx := context.Background()
	d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
	if err != nil {
		t.Fatal(err)
	}
	d.DeletePageAfter(ctx, 0)

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(deck.Fragment{}, "ClassName", "SoftLineBreak"),
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			contents, err := md.Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			fromMd := contents.ToSlides()
			d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, fromMd); err != nil {
				t.Fatal(err)
			}
			applied, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(fromMd, applied, cmpopts...); diff != "" {
				t.Errorf("diff after apply: %s", diff)
			}
			if err := d.Apply(ctx, applied); err != nil {
				t.Fatal(err)
			}
			applied2, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(applied, applied2, cmpopts...); diff != "" {
				t.Errorf("diff after re-apply: %s", diff)
			}
		})
	}
}
