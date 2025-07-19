package deck

import (
	"context"
	"fmt"

	"github.com/k1LoW/errors"
	"google.golang.org/api/slides/v1"
)

// ListSlideURLs lists URLs of the slides in the Google Slides presentation.
func (d *Deck) ListSlideURLs() []string {
	var slideURLs []string
	for _, s := range d.presentation.Slides {
		slideURLs = append(slideURLs, fmt.Sprintf("https://docs.google.com/presentation/d/%s/present?slide=id.%s", d.id, s.ObjectId))
	}
	return slideURLs
}

func (d *Deck) DumpSlides(ctx context.Context) (_ Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if err := d.refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh presentation: %w", err)
	}
	if d.presentation == nil {
		return nil, fmt.Errorf("presentation is not loaded")
	}
	layoutObjectIdMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutObjectIdMap[l.ObjectId] = l
	}
	slides := make(Slides, 0, len(d.presentation.Slides))
	for _, p := range d.presentation.Slides {
		slide := convertToSlide(p, layoutObjectIdMap)
		slides = append(slides, slide)
	}
	return slides, nil
}
