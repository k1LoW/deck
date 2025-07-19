package deck_test

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/k1LoW/deck"
	"github.com/k1LoW/deck/md"
	"github.com/lestrrat-go/backoff/v2"
)

var testCodeBlockToImageCmd = func() string {
	abs, err := filepath.Abs("testdata/txt2img/main.go")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("go run -mod=mod %s", abs)
}()

func TestApplyMarkdown(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	ctx := context.Background()

	tests := []struct {
		in string
	}{
		{"testdata/slide.md"},
		{"testdata/cap.md"},
		{"testdata/br.md"},
		{"testdata/list_simple.md"},
		{"testdata/list_and_paragraph.md"},
		{"testdata/paragraph_and_list.md"},
		{"testdata/paragraphs.md"},
		{"testdata/breaks_enabled.md"},
		{"testdata/breaks_default.md"},
		{"testdata/bold_and_italic.md"},
		{"testdata/emoji.md"},
		{"testdata/code.md"},
		{"testdata/style.md"},
		{"testdata/empty_list.md"},
		{"testdata/empty_link.md"},
		{"testdata/single_list.md"},
		{"testdata/nested_list.md"},
		{"testdata/images.md"},
		{"testdata/blockquote.md"},
		{"testdata/codeblock.md"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			// Acquire a presentation from the pool
			presentationID := deck.AcquirePresentation(t)
			defer deck.ReleasePresentation(presentationID)

			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			markdownData, err := md.Parse("testdata", b)
			if err != nil {
				t.Fatal(err)
			}
			fromMd, err := markdownData.Contents.ToSlides(ctx, testCodeBlockToImageCmd)
			if err != nil {
				t.Fatal(err)
			}
			d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, fromMd); err != nil {
				t.Fatal(err)
			}
			urls := d.ListSlideURLs()
			p := backoff.Exponential(
				backoff.WithMinInterval(time.Second),
				backoff.WithMaxInterval(5*time.Second),
				backoff.WithJitterFactor(0.05),
				backoff.WithMaxRetries(5),
			)
			for i, url := range urls {
				page := i + 1
				b := p.Start(ctx)
				if err := func() (errr error) {
					for backoff.Continue(b) {
						got := deck.Screenshot(t, url)
						p := fmt.Sprintf("%s-%d.golden.png", tt.in, page)
						if os.Getenv("UPDATE_GOLDEN") != "" {
							if err := os.WriteFile(p, got, 0600); err != nil {
								t.Fatalf("failed to update golden file %s: %v", p, err)
							}
							return nil
						}
						want, err := os.ReadFile(p)
						if err != nil {
							t.Fatalf("failed to read golden file %s: %v", p, err)
						}
						gotImage, err := png.Decode(bytes.NewReader(got))
						if err != nil {
							t.Fatalf("failed to decode screenshot: %v", err)
						}
						gotPHash, err := goimagehash.PerceptionHash(gotImage)
						if err != nil {
							t.Fatalf("failed to compute perception hash for screenshot: %v", err)
						}
						wantImage, err := png.Decode(bytes.NewReader(want))
						if err != nil {
							t.Fatalf("failed to decode golden screenshot: %v", err)
						}
						wantPHash, err := goimagehash.PerceptionHash(wantImage)
						if err != nil {
							t.Fatalf("failed to compute perception hash for golden screenshot: %v", err)
						}
						distance, err := gotPHash.Distance(wantPHash)
						if err != nil {
							t.Fatalf("failed to compute distance between screenshots: %v", err)
						}
						if distance > 1 { // threshold for similarity
							diffpath := fmt.Sprintf("%s-%d.diff.png", tt.in, page)
							if err := os.WriteFile(diffpath, got, 0600); err != nil {
								t.Fatalf("failed to write diff file %s: %v", diffpath, err)
							}
							errr = fmt.Errorf("screenshot %s does not match golden file %s: distance %d, see %s for diff", p, tt.in, distance, diffpath)
							continue
						}
						return nil
					}
					return
				}(); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestRoundTripSlidesToGoogleSlidesPresentationAndBack(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	ctx := context.Background()

	tests := []struct {
		in string
	}{
		{"testdata/slide.md"},
		{"testdata/cap.md"},
		{"testdata/br.md"},
		{"testdata/list_simple.md"},
		{"testdata/list_and_paragraph.md"},
		{"testdata/paragraph_and_list.md"},
		{"testdata/paragraphs.md"},
		//{"testdata/breaks_enabled.md"}, // FIXME: fragment merge processing is required
		//{"testdata/breaks_default.md"},
		{"testdata/bold_and_italic.md"},
		{"testdata/emoji.md"},
		{"testdata/code.md"},
		//{"testdata/style.md"},  // FIXME: class is not supported yet
		{"testdata/empty_list.md"},
		{"testdata/empty_link.md"},
		{"testdata/nested_list.md"},
		{"testdata/images.md"},
		//{"testdata/blockquote.md"},
	}

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(deck.Fragment{}, "StyleName"),
		cmpopts.IgnoreFields(deck.Slide{}, "TitleBodies", "SubtitleBodies"),
		cmpopts.IgnoreUnexported(deck.Slide{}, deck.Image{}),
		cmpopts.EquateEmpty(),
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()

			// Acquire a presentation from the pool
			presentationID := deck.AcquirePresentation(t)
			defer deck.ReleasePresentation(presentationID)

			b, err := os.ReadFile(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			markdownData, err := md.Parse("testdata", b)
			if err != nil {
				t.Fatal(err)
			}
			base, err := markdownData.Contents.ToSlides(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			d, err := deck.New(ctx, deck.WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			// Clear existing slides before applying new ones
			if err := d.DeletePageAfter(ctx, 0); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, base); err != nil {
				t.Fatal(err)
			}
			applied, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !base.Compare(applied) {
				diff := cmp.Diff(base, applied, cmpopts...)
				t.Errorf("slides after apply do not match base: %s", diff)
			}
			if err := d.Apply(ctx, applied); err != nil {
				t.Fatal(err)
			}
			applied2, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if !applied.Compare(applied2) {
				diff := cmp.Diff(applied, applied2, cmpopts...)
				t.Errorf("slides after apply do not match base: %s", diff)
			}
		})
	}
}
