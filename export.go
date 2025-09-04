package deck

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/k1LoW/errors"
	"google.golang.org/api/slides/v1"
)

// AppendPage appends a new slide to the end of the presentation.
// The deck command currently does not utilize this method and is only used within tests;
// however, it has been retained for potential future usage as a library.
func (d *Deck) AppendPage(ctx context.Context, slide *Slide) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d.logger.Info("appending new page")
	index := len(d.presentation.Slides)
	if err := d.createPage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	if reqs, err := d.prepareToApplyPage(ctx, index, slide, nil); err != nil {
		return fmt.Errorf("failed to apply page: %w", err)
	} else if len(reqs) > 0 {
		if err := d.batchUpdate(ctx, reqs); err != nil {
			return err
		}
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	d.logger.Info("appended page")
	return nil
}

// Delete deletes a Google Slides presentation by ID.
// The deck command currently does not utilize this method and is only used within tests;
// however, it has been retained for potential future usage as a library.
func Delete(ctx context.Context, id string, opts ...Option) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d, err := newDeck(ctx, opts...)
	if err != nil {
		return err
	}
	if err := d.deleteOrTrashFile(ctx, id); err != nil {
		return fmt.Errorf("failed to delete presentation: %w", err)
	}
	return nil
}

// InsertPage inserts a new slide at the specified index in the presentation.
// The deck command currently does not utilize this method and is only used within tests;
// however, it has been retained for potential future usage as a library.
func (d *Deck) InsertPage(ctx context.Context, index int, slide *Slide) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d.logger.Info("inserting page", slog.Int("index", index))
	if len(d.presentation.Slides) <= index {
		return fmt.Errorf("index out of range: %d", index)
	}
	if err := d.createPage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	if index == 0 {
		if err := d.movePage(ctx, 1, 0); err != nil {
			return fmt.Errorf("failed to move page: %w", err)
		}
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	if reqs, err := d.prepareToApplyPage(ctx, index, slide, nil); err != nil {
		return fmt.Errorf("failed to apply page: %w", err)
	} else if len(reqs) > 0 {
		if err := d.batchUpdate(ctx, reqs); err != nil {
			return err
		}
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	d.logger.Info("inserted page", slog.Int("index", index))
	return nil
}

// DumpSlides retrieves all slides from the presentation and converts them into the internal Slides structure.
// The deck command currently does not utilize this method and is only used within tests;
// however, it has been retained for potential future usage as a library.
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
