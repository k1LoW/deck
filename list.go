package deck

import (
	"context"
	"fmt"

	"github.com/k1LoW/errors"
	"google.golang.org/api/slides/v1"
)

// List Google Slides presentations.
func List(ctx context.Context, opts ...Option) (_ []*Presentation, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d := &Deck{
		styles: map[string]*slides.TextStyle{},
		shapes: map[string]*slides.ShapeProperties{},
	}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	if err := d.initialize(ctx); err != nil {
		return nil, err
	}
	return d.List()
}

// List Google Slides presentations.
func (d *Deck) List() (_ []*Presentation, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	var presentations []*Presentation

	r, err := d.driveSrv.Files.List().Q("mimeType='application/vnd.google-apps.presentation'").Fields("files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	for _, f := range r.Files {
		presentations = append(presentations, &Presentation{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	return presentations, nil
}

// ListLayouts lists layouts of the presentation.
func (d *Deck) ListLayouts() []string {
	var layouts []string
	for _, l := range d.presentation.Layouts {
		layouts = append(layouts, l.LayoutProperties.DisplayName)
	}
	return layouts
}

// ListSlideURLs lists URLs of the slides in the Google Slides presentation.
func (d *Deck) ListSlideURLs() []string {
	var slideURLs []string
	baseURL := PresentationIDtoURL(d.id)
	for _, s := range d.presentation.Slides {
		slideURLs = append(slideURLs, baseURL+"present?slide=id."+s.ObjectId)
	}
	return slideURLs
}

// PresentationIDtoURL converts a presentation ID to a Google Slides URL.
func PresentationIDtoURL(presentationID string) string {
	return fmt.Sprintf("https://docs.google.com/presentation/d/%s/", presentationID)
}
