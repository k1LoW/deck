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

func TestApplyMarkdown(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

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
		{"testdata/images.md"},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			p, err := md.New()
			if err != nil {
				t.Fatal(err)
			}
			contents, err := p.Parse("testdata", b)
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

func TestMarkdownToSlide(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	presentationID := os.Getenv("TEST_PRESENTATION_ID")

	tests := []struct {
		in string
	}{
		{"testdata/slide.md"},
		{"testdata/cap.md"},
		{"testdata/br.md"},
		{"testdata/bold_and_italic.md"},
		//{"testdata/list_and_paragraph.md"}, // FIXME: paragraph is not supported yet
		//{"testdata/paragraph_and_list.md"},  // FIXME: paragraph is not supported yet
		{"testdata/emoji.md"},
		{"testdata/code.md"},
		//{"testdata/style.md"},  // FIXME: class is not supported yet
		{"testdata/empty_list.md"},
		{"testdata/empty_link.md"},
		// {"testdata/images.md"},
	}

	ctx := context.Background()
	d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
	if err != nil {
		t.Fatal(err)
	}
	if err := d.DeletePageAfter(ctx, 0); err != nil {
		t.Fatal(err)
	}

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(deck.Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(deck.Slide{}, deck.Image{}),
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			p, err := md.New()
			if err != nil {
				t.Fatal(err)
			}
			contents, err := p.Parse("testdata", b)
			if err != nil {
				t.Fatal(err)
			}
			fromMd := contents.ToSlides()
			d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			// Clear existing slides before applying new ones
			if err := d.DeletePageAfter(ctx, 0); err != nil {
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
			for i, slide := range applied {
				for _, image := range slide.Images {
					found := false
					for _, mdImage := range fromMd[i].Images {
						if deck.CompareImages(image, mdImage) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("image not found in slide %d", i+1)
					}
				}
			}
			if err := d.Apply(ctx, applied); err != nil {
				t.Fatal(err)
			}
			applied2, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for i, slide := range applied2 {
				for _, image := range slide.Images {
					found := false
					for _, mdImage := range applied[i].Images {
						if deck.CompareImages(image, mdImage) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("image not found in slide %d", i+1)
					}
				}
			}
			if diff := cmp.Diff(applied, applied2, cmpopts...); diff != "" {
				t.Errorf("diff after re-apply: %s", diff)
			}
		})
	}
}
