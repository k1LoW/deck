package deck

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/k1LoW/deck/config"
	"github.com/k1LoW/errors"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

const layoutNameForStyle = "style"

var profileRe = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

type Deck struct {
	id                 string
	profile            string
	folderID           string
	srv                *slides.Service
	driveSrv           *drive.Service
	presentation       *slides.Presentation
	defaultTitleLayout string
	defaultLayout      string
	styles             map[string]*slides.TextStyle
	shapes             map[string]*slides.ShapeProperties
	tableStyle         *TableStyle
	logger             *slog.Logger
	fresh              bool
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

func WithProfile(profile string) Option {
	return func(d *Deck) error {
		if !profileRe.MatchString(profile) {
			return fmt.Errorf("invalid profile name: %s, only alphanumeric characters, underscores, and hyphens are allowed", profile)
		}
		d.profile = profile
		return nil
	}
}

func WithFolderID(folderID string) Option {
	return func(d *Deck) error {
		d.folderID = folderID
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
	d, err := newDeck(ctx, opts...)
	if err != nil {
		return nil, err
	}
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// Create Google Slides presentation.
func Create(ctx context.Context, opts ...Option) (_ *Deck, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d, err := newDeck(ctx, opts...)
	if err != nil {
		return nil, err
	}
	title := "Untitled"
	file := &drive.File{
		Name:     title,
		MimeType: "application/vnd.google-apps.presentation",
	}
	if d.folderID != "" {
		file.Parents = []string{d.folderID}
	}
	f, err := d.driveSrv.Files.Create(file).SupportsAllDrives(true).Do()
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
func CreateFrom(ctx context.Context, id string, opts ...Option) (_ *Deck, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	d, err := newDeck(ctx, opts...)
	if err != nil {
		return nil, err
	}
	// copy presentation
	file := &drive.File{
		Name:     "Untitled",
		MimeType: "application/vnd.google-apps.presentation",
	}
	if d.folderID != "" {
		file.Parents = []string{d.folderID}
	}
	f, err := d.driveSrv.Files.Copy(id, file).SupportsAllDrives(true).Do()
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

func Doctor(ctx context.Context, opts ...Option) error {
	d, err := newDeck(ctx, opts...)
	if err != nil {
		return err
	}
	_, err = d.getDefaultHTTPClient(ctx)
	return err
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
	if _, err := d.driveSrv.Files.Update(d.id, file).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
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

func (d *Deck) DeletePages(ctx context.Context, indices []int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	reqs := make([]*slides.Request, 0, len(indices))
	for _, idx := range indices {
		if len(d.presentation.Slides) <= idx {
			continue
		}
		currentSlide := d.presentation.Slides[idx]
		reqs = append(reqs, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: currentSlide.ObjectId,
			},
		})
	}
	if len(reqs) > 0 {
		d.logger.Info("deleting pages", slog.Any("indices", indices))
		if err := d.batchUpdate(ctx, reqs); err != nil {
			return fmt.Errorf("failed to delete pages: %w", err)
		}
		if err := d.refresh(ctx); err != nil {
			return fmt.Errorf("failed to refresh presentation after delete pages: %w", err)
		}
		d.logger.Info("deleted pages", slog.Int("count", len(reqs)), slog.Any("indices", indices))
	}
	return nil
}

func (d *Deck) DeletePageAfter(ctx context.Context, index int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if len(d.presentation.Slides) <= index+1 {
		return nil
	}
	var reqs []*slides.Request

	for i := index + 1; i < len(d.presentation.Slides); i++ {
		reqs = append(reqs, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: d.presentation.Slides[i].ObjectId,
			},
		})
	}
	if err := d.batchUpdate(ctx, reqs); err != nil {
		return err
	}
	if err := d.refresh(ctx); err != nil {
		return err
	}
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

// AllowReadingByAnyone sets the permission of the object to allow anyone to read it.
func (d *Deck) AllowReadingByAnyone(ctx context.Context, objectID string) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	permission := &drive.Permission{
		Type: "anyone",
		Role: "reader",
	}
	if _, err := d.driveSrv.Permissions.Create(objectID, permission).SupportsAllDrives(true).Context(ctx).Do(); err != nil {
		return fmt.Errorf("failed to set permission: %w", err)
	}
	return nil
}

func newDeck(ctx context.Context, opts ...Option) (*Deck, error) {
	d := &Deck{
		styles:     map[string]*slides.TextStyle{},
		shapes:     map[string]*slides.ShapeProperties{},
		tableStyle: defaultTableStyle(),
	}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	err := d.initialize(ctx)
	return d, err
}

var HTTPClientError = errors.New("http client error")

func (d *Deck) initialize(ctx context.Context) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if d.logger == nil {
		d.logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	if err := os.MkdirAll(config.StateHomePath(), 0700); err != nil {
		return err
	}

	// Get client option (service account or OAuth2)
	client, err := d.getHTTPClient(ctx)
	if err != nil {
		return errors.Join(err, HTTPClientError)
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
	reqs := []*slides.Request{{
		CreateSlide: &slides.CreateSlideRequest{
			InsertionIndex: int64(index),
			SlideLayoutReference: &slides.LayoutReference{
				LayoutId: layout.ObjectId,
			},
		},
	}}

	if err := d.batchUpdate(ctx, reqs); err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return err
	}

	return nil
}

// preparePages prepares the pages by creating slides with the specified layout IDs.
func (d *Deck) preparePages(ctx context.Context, startIdx int, layoutIDs []string) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	slideIdx := startIdx
	reqs := make([]*slides.Request, len(layoutIDs))
	for i, layoutID := range layoutIDs {
		reqs[i] = &slides.Request{
			CreateSlide: &slides.CreateSlideRequest{
				InsertionIndex: int64(slideIdx),
				SlideLayoutReference: &slides.LayoutReference{
					LayoutId: layoutID,
				},
			},
		}
		slideIdx++
	}
	if err := d.batchUpdate(ctx, reqs); err != nil {
		return err
	}
	d.logger.Debug("prepared pages", slog.Int("count", len(layoutIDs)), slog.Int("start_index", startIdx))
	return d.refresh(ctx)
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

	reqs := []*slides.Request{{
		UpdateSlidesPosition: &slides.UpdateSlidesPositionRequest{
			SlideObjectIds:  []string{currentSlide.ObjectId},
			InsertionIndex:  int64(to_index),
			ForceSendFields: []string{"InsertionIndex"},
		},
	}}
	if err := d.batchUpdate(ctx, reqs); err != nil {
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
	if d.fresh {
		return nil
	}
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
				// Extract text styles from shapes
				if e.Shape != nil && e.Shape.Text != nil {
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

				// Extract table style from 2x2 table
				if e.Table != nil {
					if ts := extractTableStyleFromLayout(e.Table); ts != nil {
						d.tableStyle = ts
					}
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
	d.fresh = true
	return nil
}

// deleteOrTrashFile attempts to delete a file, or move it to trash if deletion is not allowed.
func (d *Deck) deleteOrTrashFile(ctx context.Context, id string) error {
	file, err := d.driveSrv.Files.Get(id).SupportsAllDrives(true).Fields("capabilities").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("file not found or not accessible before deletion (file ID: %s): %w", id, err)
	}

	if file.Capabilities == nil || file.Capabilities.CanDelete {
		return d.driveSrv.Files.Delete(id).SupportsAllDrives(true).Context(ctx).Do()
	}
	if file.Capabilities.CanTrash {
		updateRequest := &drive.File{Trashed: true}
		_, err := d.driveSrv.Files.Update(id, updateRequest).SupportsAllDrives(true).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to trash presentation: %w", err)
		}
		return nil
	}
	return fmt.Errorf("file cannot be deleted or trashed (file ID: %s)", id)
}
