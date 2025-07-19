package deck

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

const (
	layoutNameForStyle             = "style"
	styleCode                      = "code"
	styleBold                      = "bold"
	styleItalic                    = "italic"
	styleLink                      = "link"
	styleBlockQuote                = "blockquote"
	defaultCodeFontFamily          = "Noto Sans Mono"
	descriptionImageFromMarkdown   = "Image generated from markdown"
	descriptionTextboxFromMarkdown = "Textbox generated from markdown"
)

type Deck struct {
	id                 string
	srv                *slides.Service
	driveSrv           *drive.Service
	presentation       *slides.Presentation
	defaultTitleLayout string
	defaultLayout      string
	styles             map[string]*slides.TextStyle
	shapes             map[string]*slides.ShapeProperties
	logger             *slog.Logger
}

type Option func(*Deck) error

func WithPresentationID(id string) Option {
	return func(d *Deck) error {
		d.id = id
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(d *Deck) error {
		d.logger = logger
		return nil
	}
}

type placeholder struct {
	objectID string
	x        float64
	y        float64
}

type bulletRange struct {
	bullet Bullet
	start  int64
	end    int64
}

type textBox struct {
	paragraphs   []*Paragraph
	fromMarkdown bool
}

type actionDetail struct {
	ActionType  actionType `json:"action_type"`
	Titles      []string   `json:"titles,omitempty"`
	Index       *int       `json:"index,omitempty"`
	MoveToIndex *int       `json:"move_to_index,omitempty"`
}

// Presentation represents a Google Slides presentation.
type Presentation struct {
	ID    string
	Title string
}

// New creates a new Deck.
func New(ctx context.Context, opts ...Option) (_ *Deck, err error) {
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
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// Create Google Slides presentation.
func Create(ctx context.Context) (_ *Deck, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d := &Deck{
		styles: map[string]*slides.TextStyle{},
		shapes: map[string]*slides.ShapeProperties{},
	}
	if err := d.initialize(ctx); err != nil {
		return nil, err
	}
	title := "Untitled"
	file := &drive.File{
		Name:     title,
		MimeType: "application/vnd.google-apps.presentation",
	}
	f, err := d.driveSrv.Files.Create(file).Do()
	if err != nil {
		return nil, err
	}
	d.id = f.Id
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// CreateFrom creates a new Deck from the presentation ID.
func CreateFrom(ctx context.Context, id string) (_ *Deck, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d := &Deck{
		styles: map[string]*slides.TextStyle{},
		shapes: map[string]*slides.ShapeProperties{},
	}
	if err := d.initialize(ctx); err != nil {
		return nil, err
	}
	// copy presentation
	file := &drive.File{
		Name:     "Untitled",
		MimeType: "application/vnd.google-apps.presentation",
	}
	f, err := d.driveSrv.Files.Copy(id, file).Do()
	if err != nil {
		return nil, err
	}
	d.id = f.Id
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	// delete all slides
	if err := d.DeletePageAfter(ctx, -1); err != nil {
		return nil, err
	}
	// create first slide
	if err := d.createPage(ctx, 0, &Slide{
		Layout: d.defaultTitleLayout,
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// Delete deletes a Google Slides presentation by ID.
func Delete(ctx context.Context, id string) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d := &Deck{
		styles: map[string]*slides.TextStyle{},
	}
	if err := d.initialize(ctx); err != nil {
		return err
	}
	if err := d.driveSrv.Files.Delete(id).Context(ctx).Do(); err != nil {
		return fmt.Errorf("failed to delete presentation: %w", err)
	}
	return nil
}

// AllowReadingByAnyone sets the permission of the presentation to allow anyone to read it.
func (d *Deck) AllowReadingByAnyone(ctx context.Context) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if d.id == "" {
		return fmt.Errorf("presentation ID is not set")
	}
	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	if _, err := d.driveSrv.Permissions.Create(d.id, permission).Context(ctx).Do(); err != nil {
		return fmt.Errorf("failed to set permission: %w", err)
	}
	return nil
}

// ID returns the ID of the presentation.
func (d *Deck) ID() string {
	return d.id
}

// UpdateTitle updates the title of the presentation.
func (d *Deck) UpdateTitle(ctx context.Context, title string) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	file := &drive.File{
		Name: title,
	}
	if _, err := d.driveSrv.Files.Update(d.id, file).Context(ctx).Do(); err != nil {
		return err
	}
	return nil
}

// Export the presentation as PDF.
func (d *Deck) Export(ctx context.Context, w io.Writer) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	req, err := d.driveSrv.Files.Export(d.id, "application/pdf").Context(ctx).Download()
	if err != nil {
		return err
	}
	if err := req.Write(w); err != nil {
		return fmt.Errorf("unable to create PDF file: %w", err)
	}
	return nil
}

func (d *Deck) DeletePage(ctx context.Context, index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d.logger.Info("deleting page", slog.Int("index", index))
	if err := d.deletePage(ctx, index); err != nil {
		return err
	}
	d.logger.Info("deleted page", slog.Int("index", index))
	return nil
}

func (d *Deck) DeletePageAfter(ctx context.Context, index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if len(d.presentation.Slides) <= index {
		return nil
	}
	req := &slides.BatchUpdatePresentationRequest{}
	for i := index + 1; i < len(d.presentation.Slides); i++ {
		req.Requests = append(req.Requests, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: d.presentation.Slides[i].ObjectId,
			},
		})
	}
	if len(req.Requests) == 0 {
		return nil
	}
	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
		return err
	}
	if err := d.refresh(ctx); err != nil {
		return err
	}
	return nil
}

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
	if err := d.applyPage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to apply page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	d.logger.Info("appended page")
	return nil
}

func (d *Deck) MovePage(ctx context.Context, from_index, to_index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d.logger.Info("moving page", slog.Int("from_index", from_index), slog.Int("to_index", to_index))
	if err := d.movePage(ctx, from_index, to_index); err != nil {
		return err
	}
	d.logger.Info("moved page", slog.Int("from_index", from_index), slog.Int("to_index", to_index))
	return nil
}

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
	if err := d.applyPage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to apply page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	d.logger.Info("inserted page", slog.Int("index", index))
	return nil
}

func (d *Deck) initialize(ctx context.Context) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if d.logger == nil {
		d.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	if err := os.MkdirAll(config.DataHomePath(), 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(config.StateHomePath(), 0700); err != nil {
		return err
	}

	creds := filepath.Join(config.DataHomePath(), "credentials.json")
	b, err := os.ReadFile(creds)
	if err != nil {
		return err
	}

	config, err := google.ConfigFromJSON(b, slides.PresentationsScope, slides.DriveScope)
	if err != nil {
		return err
	}

	client, err := d.getHTTPClient(ctx, config)
	if err != nil {
		return err
	}
	srv, err := slides.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	srv.UserAgent = userAgent
	d.srv = srv
	driveSrv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}
	driveSrv.UserAgent = userAgent
	d.driveSrv = driveSrv
	return nil
}

func (d *Deck) createPage(ctx context.Context, index int, slide *Slide) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	layoutMap := d.layoutMap()
	layout, ok := layoutMap[slide.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %q", slide.Layout)
	}

	// create new page
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				CreateSlide: &slides.CreateSlideRequest{
					InsertionIndex: int64(index),
					SlideLayoutReference: &slides.LayoutReference{
						LayoutId: layout.ObjectId,
					},
				},
			},
		},
	}

	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
		return err
	}

	if err := d.refresh(ctx); err != nil {
		return err
	}

	return nil
}

func (d *Deck) deletePage(ctx context.Context, index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if len(d.presentation.Slides) <= index {
		return nil
	}
	currentSlide := d.presentation.Slides[index]
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: currentSlide.ObjectId,
				},
			},
		},
	}
	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
		return err
	}
	if err := d.refresh(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Deck) movePage(ctx context.Context, from_index, to_index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if from_index == to_index || from_index < 0 || to_index < 0 || from_index >= len(d.presentation.Slides) || to_index >= len(d.presentation.Slides) {
		return nil
	}

	currentSlide := d.presentation.Slides[from_index]

	if from_index < to_index {
		to_index++
	}

	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateSlidesPosition: &slides.UpdateSlidesPositionRequest{
					SlideObjectIds:  []string{currentSlide.ObjectId},
					InsertionIndex:  int64(to_index),
					ForceSendFields: []string{"InsertionIndex"},
				},
			},
		},
	}

	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
		return err
	}
	if err := d.refresh(ctx); err != nil {
		return err
	}
	return nil
}

func (d *Deck) layoutMap() map[string]*slides.Page {
	layoutMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutMap[l.LayoutProperties.DisplayName] = l
	}
	return layoutMap
}

func (d *Deck) refresh(ctx context.Context) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	presentation, err := d.srv.Presentations.Get(d.id).Context(ctx).Do()
	if err != nil {
		return err
	}
	d.presentation = presentation

	// set default layouts and detect style
	for _, l := range d.presentation.Layouts {
		layout := l.LayoutProperties.Name
		switch {
		case strings.HasPrefix(layout, "TITLE_AND_BODY"):
			if d.defaultLayout == "" {
				d.defaultLayout = l.LayoutProperties.DisplayName
			}
		case strings.HasPrefix(layout, "TITLE"):
			if d.defaultTitleLayout == "" {
				d.defaultTitleLayout = l.LayoutProperties.DisplayName
			}
		}

		if l.LayoutProperties.DisplayName == layoutNameForStyle {
			for _, e := range l.PageElements {
				if e.Shape == nil || e.Shape.Text == nil {
					continue
				}
				for _, t := range e.Shape.Text.TextElements {
					if t.TextRun == nil {
						continue
					}
					styleName := strings.Trim(t.TextRun.Content, " \n")
					if styleName == "" {
						continue
					}
					d.styles[styleName] = t.TextRun.Style
					d.shapes[styleName] = e.Shape.ShapeProperties
				}
			}
		}
	}

	// If the default layouts that were derived are renamed or otherwise disappear, search for them again.
	// The defaultLayout may be an empty string, but even in that case, the layout search from the map
	// will fail, so this case is also covered.
	layoutMap := d.layoutMap()
	_, defaultTitleLayoutFound := layoutMap[d.defaultTitleLayout]
	_, defaultLayoutFound := layoutMap[d.defaultLayout]

	if !defaultTitleLayoutFound {
		d.defaultTitleLayout = d.presentation.Layouts[0].LayoutProperties.DisplayName
	}
	if !defaultLayoutFound {
		if len(d.presentation.Layouts) > 1 {
			d.defaultLayout = d.presentation.Layouts[1].LayoutProperties.DisplayName
		} else {
			d.defaultLayout = d.presentation.Layouts[0].LayoutProperties.DisplayName
		}
	}

	return nil
}
