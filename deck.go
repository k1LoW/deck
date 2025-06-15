package deck

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

const (
	layoutNameForStyle    = "style"
	styleCode             = "code"
	styleBold             = "bold"
	styleItalic           = "italic"
	styleLink             = "link"
	defaultCodeFontFamily = "Noto Sans Mono"
)

type Slides []*Slide

type Slide struct {
	Layout      string   `json:"layout"`
	Freeze      bool     `json:"freeze,omitempty"`
	Titles      []string `json:"titles,omitempty"`
	Subtitles   []string `json:"subtitles,omitempty"`
	Bodies      []*Body  `json:"bodies,omitempty"`
	SpeakerNote string   `json:"speakerNote,omitempty"`

	new    bool
	delete bool
}

// Body represents the content body of a slide.
type Body struct {
	Paragraphs []*Paragraph `json:"paragraphs,omitempty"`
}

// Paragraph represents a paragraph within a slide body.
type Paragraph struct {
	Fragments []*Fragment `json:"fragments,omitempty"`
	Bullet    Bullet      `json:"bullet,omitempty"`
	Nesting   int         `json:"nesting,omitempty"`
}

// Fragment represents a text fragment within a paragraph.
type Fragment struct {
	Value         string `json:"value"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Link          string `json:"link,omitempty"`
	Code          bool   `json:"code,omitempty"`
	SoftLineBreak bool   `json:"softLineBreak,omitempty"`
	ClassName     string `json:"className,omitempty"`
}

// Bullet represents the type of bullet point for a paragraph.
type Bullet string

// Bullet constants for different bullet point types.
const (
	BulletNone   Bullet = ""
	BulletDash   Bullet = "-"
	BulletNumber Bullet = "1"
	BulletAlpha  Bullet = "a"
)

type Deck struct {
	id                   string
	dataHomePath         string
	stateHomePath        string
	srv                  *slides.Service
	driveSrv             *drive.Service
	presentation         *slides.Presentation
	defaultTitleLayout   string
	defaultSectionLayout string
	defaultLayout        string
	styles               map[string]*slides.TextStyle
	logger               *slog.Logger
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
	start  int
	end    int
}

// Presentation represents a Google Slides presentation.
type Presentation struct {
	ID    string
	Title string
}

// New creates a new Deck.
func New(ctx context.Context, opts ...Option) (*Deck, error) {
	d, err := initialize(ctx)
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		if err := opt(d); err != nil {
			return nil, err
		}
	}
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	return d, nil
}

// Create Google Slides presentation.
func Create(ctx context.Context) (*Deck, error) {
	d, err := initialize(ctx)
	if err != nil {
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
func CreateFrom(ctx context.Context, id string) (*Deck, error) {
	d, err := initialize(ctx)
	if err != nil {
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
	if err := d.CreatePage(ctx, 0, &Slide{
		Layout: d.defaultTitleLayout,
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// List Google Slides presentations.
func List(ctx context.Context) ([]*Presentation, error) {
	d, err := initialize(ctx)
	if err != nil {
		return nil, err
	}
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

func initialize(ctx context.Context) (*Deck, error) {
	d := &Deck{
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		styles: map[string]*slides.TextStyle{},
	}
	if os.Getenv("XDG_DATA_HOME") != "" {
		d.dataHomePath = filepath.Join(os.Getenv("XDG_DATA_HOME"), "deck")
	} else {
		d.dataHomePath = filepath.Join(os.Getenv("HOME"), ".local", "share", "deck")
	}
	if os.Getenv("XDG_STATE_HOME") != "" {
		d.stateHomePath = filepath.Join(os.Getenv("XDG_STATE_HOME"), "deck")
	} else {
		d.stateHomePath = filepath.Join(os.Getenv("HOME"), ".local", "state", "deck")
	}
	if err := os.MkdirAll(d.dataHomePath, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(d.stateHomePath, 0700); err != nil {
		return nil, err
	}

	creds := filepath.Join(d.dataHomePath, "credentials.json")
	b, err := os.ReadFile(creds)
	if err != nil {
		return nil, err
	}

	config, err := google.ConfigFromJSON(b, slides.PresentationsScope, slides.DriveScope)
	if err != nil {
		return nil, err
	}

	client, err := d.getHTTPClient(ctx, config)
	if err != nil {
		return nil, err
	}
	srv, err := slides.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	d.srv = srv
	driveSrv, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	d.driveSrv = driveSrv
	return d, nil
}

// ID returns the ID of the presentation.
func (d *Deck) ID() string {
	return d.id
}

// List Google Slides presentations.
func (d *Deck) List() ([]*Presentation, error) {
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

// Apply the markdown slides to the presentation.
func (d *Deck) Apply(ctx context.Context, slides Slides) error {
	pages := make([]int, 0, len(slides))
	for i := range len(slides) {
		pages = append(pages, i+1)
	}
	return d.ApplyPages(ctx, slides, pages)
}

// ApplyPages applies the markdown slides to the presentation with the specified pages.
func (d *Deck) ApplyPages(ctx context.Context, ss Slides, pages []int) error {
	layoutObjectIdMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutObjectIdMap[l.ObjectId] = l
	}

	before := make(Slides, 0, len(d.presentation.Slides))
	var after Slides
	for _, p := range d.presentation.Slides {
		slide := convertToSlide(p, layoutObjectIdMap)
		before = append(before, slide)
		after = append(after, slide)
	}

	for i, slide := range ss {
		if !slices.Contains(pages, i+1) {
			continue
		}
		if slide.Layout == "" {
			switch {
			case i == 0:
				slide.Layout = d.defaultTitleLayout
			case len(slide.Bodies) == 0:
				slide.Layout = d.defaultSectionLayout
			default:
				slide.Layout = d.defaultLayout
			}
		}
		if len(after) < i {
			after[i] = slide
		} else if len(after) == i {
			after = append(after, slide)
		} else {
			after[i] = slide
		}
	}

	actions, err := generateActions(before, after)
	if err != nil {
		return fmt.Errorf("failed to diff slides: %w", err)
	}

	slog.Info("len slides", slog.Int("len", len(d.presentation.Slides)))
	for _, action := range actions {
		var slideTitle string
		if action.slide != nil && len(action.slide.Titles) > 0 {
			slideTitle = action.slide.Titles[0]
		}
		slog.Info("action", slog.String("type", action.actionType.String()), slog.Int("index", action.index), slog.Int("moveToIndex", action.moveToIndex), slog.String("slide", slideTitle))
		switch action.actionType {
		case actionTypeAppend:
			if err := d.appendPage(ctx, action.slide); err != nil {
				return fmt.Errorf("failed to append slide: %w", err)
			}
		case actionTypeInsert:
			if err := d.insertPage(ctx, action.index, action.slide); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			}
		case actionTypeUpdate:
			if err := d.applyPage(ctx, action.index, action.slide); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			}
		case actionTypeMove:
			if err := d.movePage(ctx, action.index, action.moveToIndex); err != nil {
				return fmt.Errorf("failed to move page: %w", err)
			}
		case actionTypeDelete:
			if err := d.DeletePage(ctx, action.index); err != nil {
				return fmt.Errorf("failed to delete page: %w", err)
			}
		}
	}

	// Note: DeletePageAfter is still needed to handle cases where slides are reduced
	// but not explicitly deleted through diff actions (e.g., when the new slide count is less)
	if err := d.DeletePageAfter(ctx, len(ss)-1); err != nil {
		return err
	}

	return nil
}

// UpdateTitle updates the title of the presentation.
func (d *Deck) UpdateTitle(ctx context.Context, title string) error {
	file := &drive.File{
		Name: title,
	}
	if _, err := d.driveSrv.Files.Update(d.id, file).Context(ctx).Do(); err != nil {
		return err
	}
	return nil
}

// Export the presentation as PDF.
func (d *Deck) Export(ctx context.Context, w io.Writer) error {
	req, err := d.driveSrv.Files.Export(d.id, "application/pdf").Context(ctx).Download()
	if err != nil {
		return err
	}
	if err := req.Write(w); err != nil {
		return fmt.Errorf("unable to create PDF file: %w", err)
	}
	return nil
}

func (d *Deck) DumpSlides(ctx context.Context) (Slides, error) {
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

func (d *Deck) applyPage(ctx context.Context, index int, slide *Slide) error {
	d.logger.Info("appling page", slog.Int("index", index))
	layoutMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutMap[l.LayoutProperties.DisplayName] = l
	}

	layout, ok := layoutMap[slide.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %s", slide.Layout)
	}

	if len(d.presentation.Slides) <= index {
		// create new page
		if slide.Layout == "" {
			switch {
			case index == 0:
				slide.Layout = d.defaultTitleLayout
			case len(slide.Bodies) == 0:
				slide.Layout = d.defaultSectionLayout
			default:
				slide.Layout = d.defaultLayout
			}
		}
		if err := d.CreatePage(ctx, index, slide); err != nil {
			return err
		}
	}
	if slide.Freeze {
		d.logger.Info("skip applying page. because freeze:true", slog.Int("index", index))
		return nil
	}
	currentSlide := d.presentation.Slides[index]
	if currentSlide.SlideProperties.LayoutObjectId != layout.ObjectId {
		// create new page
		if err := d.CreatePage(ctx, index+1, slide); err != nil {
			return err
		}
		if err := d.DeletePage(ctx, index); err != nil {
			return err
		}
	}

	var (
		titles    []placeholder
		subtitles []placeholder
		bodies    []placeholder
	)
	currentSlide = d.presentation.Slides[index]
	for _, element := range currentSlide.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			switch element.Shape.Placeholder.Type {
			case "CENTERED_TITLE", "TITLE":
				titles = append(titles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(ctx, element.ObjectId); err != nil {
					return err
				}
			case "SUBTITLE":
				subtitles = append(subtitles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(ctx, element.ObjectId); err != nil {
					return err
				}
			case "BODY":
				bodies = append(bodies, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(ctx, element.ObjectId); err != nil {
					return err
				}
			}
		}
	}
	var speakerNotesID string
	for _, element := range currentSlide.SlideProperties.NotesPage.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			if element.Shape.Placeholder.Type == "BODY" {
				speakerNotesID = element.ObjectId
				if err := d.clearPlaceholder(ctx, speakerNotesID); err != nil {
					return err
				}
			}
		}
	}
	if speakerNotesID == "" {
		return fmt.Errorf("speaker notes not found")
	}

	// set titles
	req := &slides.BatchUpdatePresentationRequest{}
	sort.Slice(titles, func(i, j int) bool {
		if titles[i].y == titles[j].y {
			return titles[i].x < titles[j].x
		}
		return titles[i].y < titles[j].y
	})
	for i, title := range slide.Titles {
		if len(titles) <= i {
			continue
		}
		req.Requests = append(req.Requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: titles[i].objectID,
				Text:     title,
			},
		})
	}

	// set subtitles
	sort.Slice(subtitles, func(i, j int) bool {
		if subtitles[i].y == subtitles[j].y {
			return subtitles[i].x < subtitles[j].x
		}
		return subtitles[i].y < subtitles[j].y
	})
	for i, subtitle := range slide.Subtitles {
		if len(subtitles) <= i {
			continue
		}
		req.Requests = append(req.Requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: subtitles[i].objectID,
				Text:     subtitle,
			},
		})
	}

	// set speaker notes
	req.Requests = append(req.Requests, &slides.Request{
		InsertText: &slides.InsertTextRequest{
			ObjectId: speakerNotesID,
			Text:     slide.SpeakerNote,
		},
	})

	// set bodies
	sort.Slice(bodies, func(i, j int) bool {
		if bodies[i].y == bodies[j].y {
			return bodies[i].x < bodies[j].x
		}
		return bodies[i].y < bodies[j].y
	})
	var bulletStartIndex, bulletEndIndex int
	bulletRanges := map[int]*bulletRange{}
	for i, body := range slide.Bodies {
		if len(bodies) <= i {
			continue
		}
		count := 0
		text := ""
		bulletStartIndex = 0 // reset per body
		bulletEndIndex = 0   // reset per body
		var styleReqs []*slides.Request
		currentBullet := BulletNone
		for j, paragraph := range body.Paragraphs {
			plen := 0
			if paragraph.Bullet != BulletNone {
				if paragraph.Nesting > 0 {
					text += "\t"
					plen++
				}
			}
			for _, fragment := range paragraph.Fragments {
				flen := countString(fragment.Value)
				startIndex := ptrInt64(int64(count + plen))
				endIndex := ptrInt64(int64(count + plen + flen))

				// code
				if fragment.Code {
					s, ok := d.styles[styleCode]
					if ok {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold:            s.Bold,
									Italic:          s.Italic,
									Underline:       s.Underline,
									ForegroundColor: s.ForegroundColor,
									FontFamily:      s.FontFamily,
									BackgroundColor: s.BackgroundColor,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
							},
						})
					} else {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									ForegroundColor: &slides.OptionalColor{
										OpaqueColor: &slides.OpaqueColor{
											RgbColor: &slides.RgbColor{
												Red:   0.0,
												Green: 0.0,
												Blue:  0.0,
											},
										},
									},
									FontFamily: defaultCodeFontFamily,
									BackgroundColor: &slides.OptionalColor{
										OpaqueColor: &slides.OpaqueColor{
											RgbColor: &slides.RgbColor{
												Red:   0.95,
												Green: 0.95,
												Blue:  0.95,
											},
										},
									},
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "foregroundColor,fontFamily,backgroundColor",
							},
						})
					}
				}

				// bold
				if fragment.Bold {
					s, ok := d.styles[styleBold]
					if ok {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold:            s.Bold,
									Italic:          s.Italic,
									Underline:       s.Underline,
									ForegroundColor: s.ForegroundColor,
									FontFamily:      s.FontFamily,
									BackgroundColor: s.BackgroundColor,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
							},
						})
					} else {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold: true,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "bold",
							},
						})
					}
				}

				// italic
				if fragment.Italic {
					s, ok := d.styles[styleItalic]
					if ok {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold:            s.Bold,
									Italic:          s.Italic,
									Underline:       s.Underline,
									ForegroundColor: s.ForegroundColor,
									FontFamily:      s.FontFamily,
									BackgroundColor: s.BackgroundColor,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
							},
						})
					} else {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Italic: true,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "italic",
							},
						})
					}
				}

				// link
				if fragment.Link != "" {
					s, ok := d.styles[styleLink]
					if ok {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold:            s.Bold,
									Italic:          s.Italic,
									Underline:       s.Underline,
									ForegroundColor: s.ForegroundColor,
									FontFamily:      s.FontFamily,
									BackgroundColor: s.BackgroundColor,
									Link: &slides.Link{
										Url: fragment.Link,
									},
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "link,bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
							},
						})
					} else {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Link: &slides.Link{
										Url: fragment.Link,
									},
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "link",
							},
						})
					}
				}

				if fragment.ClassName != "" {
					s, ok := d.styles[fragment.ClassName]
					if ok {
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: bodies[i].objectID,
								Style: &slides.TextStyle{
									Bold:            s.Bold,
									Italic:          s.Italic,
									Underline:       s.Underline,
									ForegroundColor: s.ForegroundColor,
									FontFamily:      s.FontFamily,
									BackgroundColor: s.BackgroundColor,
								},
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: startIndex,
									EndIndex:   endIndex,
								},
								Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
							},
						})
					}
				}

				plen += flen
				text += fragment.Value
				if fragment.SoftLineBreak {
					text += "\n"
					plen++
				}
			}

			if len(body.Paragraphs) > j+1 {
				nextParagraph := body.Paragraphs[j+1]
				if paragraph.Bullet != nextParagraph.Bullet || paragraph.Bullet != BulletNone {
					text += "\n"
					plen++
				}
			}

			if paragraph.Bullet != BulletNone {
				if paragraph.Nesting == 0 && currentBullet != paragraph.Bullet {
					bulletStartIndex = count
					bulletEndIndex = count
					bulletRanges[bulletStartIndex] = &bulletRange{
						bullet: paragraph.Bullet,
						start:  bulletStartIndex,
						end:    bulletEndIndex,
					}
				}
				bulletEndIndex += plen
				bulletRanges[bulletStartIndex].end = bulletEndIndex
			}
			currentBullet = paragraph.Bullet
			count += plen
		}

		req.Requests = append(req.Requests, &slides.Request{
			InsertText: &slides.InsertTextRequest{
				ObjectId: bodies[i].objectID,
				Text:     text,
			},
		})
		req.Requests = append(req.Requests, styleReqs...)
		bulletRangeSlice := []*bulletRange{}
		for _, r := range bulletRanges {
			bulletRangeSlice = append(bulletRangeSlice, r)
		}
		// reverse sort
		// Because the Range changes each time it is converted to a list, convert from the end to a list.
		sort.Slice(bulletRangeSlice, func(i, j int) bool {
			return bulletRangeSlice[i].start > bulletRangeSlice[j].start
		})
		for _, r := range bulletRangeSlice {
			startIndex := int64(r.start)
			endIndex := int64(r.end - 1)
			if startIndex == endIndex {
				endIndex++
			}
			req.Requests = append(req.Requests, &slides.Request{
				CreateParagraphBullets: &slides.CreateParagraphBulletsRequest{
					ObjectId:     bodies[i].objectID,
					BulletPreset: convertBullet(r.bullet),
					TextRange: &slides.Range{
						Type:       "FIXED_RANGE",
						StartIndex: ptrInt64(startIndex),
						EndIndex:   ptrInt64(endIndex),
					},
				},
			})
		}
	}

	if len(req.Requests) > 0 {
		if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
			return err
		}
	}

	d.logger.Info("applied page", slog.Int("index", index))
	return nil
}

func (d *Deck) CreatePage(ctx context.Context, index int, slide *Slide) error {
	layoutMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutMap[l.LayoutProperties.DisplayName] = l
	}

	layout, ok := layoutMap[slide.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %s", slide.Layout)
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

func (d *Deck) DeletePage(ctx context.Context, index int) error {
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

func (d *Deck) DeletePageAfter(ctx context.Context, index int) error {
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

func (d *Deck) appendPage(ctx context.Context, slide *Slide) error {
	index := len(d.presentation.Slides)
	if err := d.CreatePage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	slog.Info("appendPage: apply page", slog.Int("index", index), slog.String("layout", slide.Layout))
	if err := d.applyPage(ctx, index, slide); err != nil {
		return fmt.Errorf("failed to apply page: %w", err)
	}
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	return nil
}

func (d *Deck) insertPage(ctx context.Context, index int, slide *Slide) error {
	if len(d.presentation.Slides) <= index {
		return fmt.Errorf("index out of range: %d", index)
	}
	if err := d.CreatePage(ctx, index, slide); err != nil {
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
	return nil
}

func (d *Deck) movePage(ctx context.Context, fromIndex, toIndex int) error {
	if fromIndex == toIndex || fromIndex < 0 || toIndex < 0 || fromIndex >= len(d.presentation.Slides) || toIndex >= len(d.presentation.Slides) {
		return nil
	}

	currentSlide := d.presentation.Slides[fromIndex]

	if fromIndex < toIndex {
		toIndex++
	}

	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateSlidesPosition: &slides.UpdateSlidesPositionRequest{
					SlideObjectIds:  []string{currentSlide.ObjectId},
					InsertionIndex:  int64(toIndex),
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

func (d *Deck) refresh(ctx context.Context) error {
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
		case strings.HasPrefix(layout, "SECTION_HEADER"):
			if d.defaultSectionLayout == "" {
				d.defaultSectionLayout = l.LayoutProperties.DisplayName
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
					className := strings.Trim(t.TextRun.Content, " \n")
					if className == "" {
						continue
					}
					d.styles[className] = t.TextRun.Style
				}
			}
		}
	}

	return nil
}

func (d *Deck) clearPlaceholder(ctx context.Context, placeholderID string) error {
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{
			{
				UpdateTextStyle: &slides.UpdateTextStyleRequest{
					ObjectId: placeholderID,
					Style: &slides.TextStyle{
						Bold:   false,
						Italic: false,
					},
					TextRange: &slides.Range{
						Type: "ALL",
					},
					Fields: "*",
				},
			},
			{
				DeleteParagraphBullets: &slides.DeleteParagraphBulletsRequest{
					ObjectId: placeholderID,
					TextRange: &slides.Range{
						Type: "ALL",
					},
				},
			},
			{
				DeleteText: &slides.DeleteTextRequest{
					ObjectId: placeholderID,
					TextRange: &slides.Range{
						Type: "ALL",
					},
				},
			},
		},
	}

	_, _ = d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do()
	return nil
}

func (d *Deck) getHTTPClient(ctx context.Context, config *oauth2.Config) (*http.Client, error) {
	tokenPath := filepath.Join(d.stateHomePath, "token.json")
	token, err := d.tokenFromFile(tokenPath)
	if err != nil {
		token, err = d.getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := d.saveToken(tokenPath, token); err != nil {
			return nil, err
		}
	} else if token.Expiry.Before(time.Now()) {
		// Token has expired, refresh it using the refresh token
		d.logger.Info("token has expired, refreshing")
		if token.RefreshToken != "" {
			tokenSource := config.TokenSource(ctx, token)
			newToken, err := tokenSource.Token()
			if err != nil {
				d.logger.Info("failed to refresh token, getting new token from web", slog.String("error", err.Error()))
				// If refresh fails, get a new token from the web
				newToken, err = d.getTokenFromWeb(ctx, config)
				if err != nil {
					return nil, err
				}
			} else {
				d.logger.Info("token refreshed successfully")
			}

			// Save the new token
			if err := d.saveToken(tokenPath, newToken); err != nil {
				return nil, err
			}
			token = newToken
		} else {
			// No refresh token available, get a new token from the web
			d.logger.Info("no refresh token available, getting new token from web")
			token, err = d.getTokenFromWeb(ctx, config)
			if err != nil {
				return nil, err
			}
			if err := d.saveToken(tokenPath, token); err != nil {
				return nil, err
			}
		}
	}
	client := config.Client(ctx, token)

	retryClient := retryablehttp.NewClient()
	retryClient.HTTPClient = client
	retryClient.RetryMax = 10
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 30 * time.Second
	retryClient.Logger = nil

	return retryClient.StandardClient(), nil
}

func (d *Deck) getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Generate code verifier and challenge for PKCE
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	var authCode string

	// Generate random state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)
	listenCtx, listening := context.WithCancel(ctx)
	doneCtx, done := context.WithCancel(ctx)
	// run and stop local server
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("code") == "" {
			return
		}
		authCode = r.URL.Query().Get("code")
		_, _ = w.Write([]byte("Received code. You may now close this tab."))
		done()
	})
	srv := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	var listenErr error
	go func() {
		ln, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			listenErr = fmt.Errorf("listen: %w", err)
			listening()
			done()
			return
		}
		srv.Addr = ln.Addr().String()
		listening()
		if err := srv.Serve(ln); err != nil {
			if err != http.ErrServerClosed {
				listenErr = fmt.Errorf("serve: %w", err)
				done()
				return
			}
		}
	}()
	<-listenCtx.Done()
	if listenErr != nil {
		return nil, listenErr
	}
	config.RedirectURL = "http://" + srv.Addr + "/"

	// Add PKCE parameters to the authorization URL
	authURL := config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"))

	if err := browser.OpenURL(authURL); err != nil {
		return nil, err
	}

	<-doneCtx.Done()
	if err := srv.Shutdown(ctx); err != nil {
		return nil, err
	}

	// Send code verifier during token exchange
	token, err := config.Exchange(ctx, authCode,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (d *Deck) tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	token := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(token); err != nil {
		return nil, err
	}
	return token, err
}

func (d *Deck) saveToken(path string, token *oauth2.Token) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("unable to cache oauth token: %w", err)
	}
	return nil
}

// generateCodeVerifier generates a code verifier for PKCE.
// Generates a random string of 43-128 characters in compliance with RFC7636.
func generateCodeVerifier() (string, error) {
	// Generate 64 bytes (512 bits) of random data
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge generates a code challenge from the code verifier.
// Calculates SHA-256 hash and applies Base64 URL encoding.
func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// countString counts the number of characters in a string, considering UTF-16 surrogate pairs.
// This is because Google Slides' character count is derived from JavaScript.
func countString(s string) int {
	length := 0
	for _, r := range s {
		if r <= 0xFFFF && (r < 0xD800 || r > 0xDFFF) {
			length++
		} else {
			length += 2
		}
	}
	return length
}

func ptrInt64(i int64) *int64 {
	return &i
}

func convertBullet(b Bullet) string {
	switch b {
	case BulletDash:
		return "BULLET_DISC_CIRCLE_SQUARE"
	case BulletNumber:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	case BulletAlpha:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	default:
		return "UNRECOGNIZED"
	}
}

func convertToSlide(p *slides.Page, layoutObjectIdMap map[string]*slides.Page) *Slide {
	slide := &Slide{
		Layout: "",
		Freeze: false,
	}
	if p.SlideProperties != nil {
		page, ok := layoutObjectIdMap[p.SlideProperties.LayoutObjectId]
		if ok {
			slide.Layout = page.LayoutProperties.DisplayName
		}
	}

	var titles []string
	var subtitles []string
	var bodies []*Body

	// Extract titles, subtitles, and bodies from page elements
	for _, element := range p.PageElements {
		if element.Shape != nil && element.Shape.Text != nil && element.Shape.Placeholder != nil {
			switch element.Shape.Placeholder.Type {
			case "CENTERED_TITLE", "TITLE":
				text := extractText(element.Shape.Text)
				if text != "" {
					titles = append(titles, text)
				}
			case "SUBTITLE":
				text := extractText(element.Shape.Text)
				if text != "" {
					subtitles = append(subtitles, text)
				}
			case "BODY":
				body := convertToBody(element.Shape.Text)
				if body != nil {
					bodies = append(bodies, body)
				}
			}
		}
	}

	slide.Titles = titles
	slide.Subtitles = subtitles
	slide.Bodies = bodies

	// Extract speaker notes
	if p.SlideProperties != nil && p.SlideProperties.NotesPage != nil {
		for _, element := range p.SlideProperties.NotesPage.PageElements {
			if element.Shape != nil && element.Shape.Text != nil && element.Shape.Placeholder != nil {
				if element.Shape.Placeholder.Type == "BODY" {
					slide.SpeakerNote = extractText(element.Shape.Text)
					break
				}
			}
		}
	}

	return slide
}

// extractText extracts plain text from Shape.Text.
func extractText(text *slides.TextContent) string {
	if text == nil || len(text.TextElements) == 0 {
		return ""
	}

	var result strings.Builder
	for _, element := range text.TextElements {
		if element.TextRun != nil {
			result.WriteString(element.TextRun.Content)
		}
	}
	return strings.TrimSpace(result.String())
}

// convertToBody generates a Body struct from Shape.Text.
func convertToBody(text *slides.TextContent) *Body {
	if text == nil || len(text.TextElements) == 0 {
		return nil
	}

	body := &Body{
		Paragraphs: []*Paragraph{},
	}

	var currentParagraph *Paragraph
	var currentBullet Bullet

	for _, element := range text.TextElements {
		if element.ParagraphMarker != nil {
			// Start of a new paragraph
			if currentParagraph != nil && len(currentParagraph.Fragments) > 0 {
				// Check if this is a continuation of a non-bullet paragraph
				// If the previous paragraph had no bullet and this one also has no bullet,
				// merge them with a newline fragment
				if currentParagraph.Bullet == BulletNone &&
					(element.ParagraphMarker.Bullet == nil) {
					// Add newline fragment to continue the paragraph
					currentParagraph.Fragments = append(currentParagraph.Fragments, &Fragment{
						Value: "\n",
					})
					// Don't create a new paragraph, continue with the current one
				} else {
					body.Paragraphs = append(body.Paragraphs, currentParagraph)
					currentParagraph = &Paragraph{
						Fragments: []*Fragment{},
						Nesting:   0,
					}
				}
			} else {
				currentParagraph = &Paragraph{
					Fragments: []*Fragment{},
					Nesting:   0,
				}
			}

			// Process bullet points
			if element.ParagraphMarker.Bullet != nil {
				// Determine the type of bullet points based on glyph content
				if element.ParagraphMarker.Bullet.Glyph != "" {
					glyph := element.ParagraphMarker.Bullet.Glyph
					// Check for numbered bullets (1, 2, 3, etc.)
					if strings.Contains(glyph, "1") || strings.Contains(glyph, "2") || strings.Contains(glyph, "3") ||
						strings.Contains(glyph, "4") || strings.Contains(glyph, "5") || strings.Contains(glyph, "6") ||
						strings.Contains(glyph, "7") || strings.Contains(glyph, "8") || strings.Contains(glyph, "9") ||
						strings.Contains(glyph, "0") {
						currentBullet = BulletNumber
					} else {
						currentBullet = BulletDash
					}
				} else {
					// If no glyph, assume it's a dash bullet
					currentBullet = BulletDash
				}
				currentParagraph.Bullet = currentBullet

				// Set nesting level
				currentParagraph.Nesting = int(element.ParagraphMarker.Bullet.NestingLevel)
			} else {
				currentBullet = BulletNone
				currentParagraph.Bullet = currentBullet
			}
		}

		if element.TextRun != nil && currentParagraph != nil {
			// Process text content
			content := element.TextRun.Content

			// Check if this is an empty content that should be treated as SoftLineBreak
			if content == "" {
				fragment := &Fragment{
					Value:         "",
					SoftLineBreak: true,
					ClassName:     "",
				}
				currentParagraph.Fragments = append(currentParagraph.Fragments, fragment)
				continue
			}

			// Handle special case where content is just a newline
			if content == "\n" {
				// Check if the previous fragment exists and can be marked with SoftLineBreak
				if len(currentParagraph.Fragments) > 0 {
					lastFragment := currentParagraph.Fragments[len(currentParagraph.Fragments)-1]
					if lastFragment.Value != "" && !lastFragment.SoftLineBreak {
						lastFragment.SoftLineBreak = true
						continue
					}
				}
				// If no previous fragment or it already has SoftLineBreak, add as newline fragment
				currentParagraph.Fragments = append(currentParagraph.Fragments, &Fragment{
					Value: "\n",
				})
				continue
			}

			// Get styles from TextRun
			var bold, italic, code bool
			var link string
			if element.TextRun.Style != nil {
				bold = element.TextRun.Style.Bold
				italic = element.TextRun.Style.Italic
				if element.TextRun.Style.Link != nil && element.TextRun.Style.Link.Url != "" {
					link = element.TextRun.Style.Link.Url
				}

				// Detect code style (based on font family and background color)
				if element.TextRun.Style.FontFamily == defaultCodeFontFamily ||
					(element.TextRun.Style.BackgroundColor != nil &&
						element.TextRun.Style.BackgroundColor.OpaqueColor != nil &&
						element.TextRun.Style.BackgroundColor.OpaqueColor.RgbColor != nil) {
					code = true
				}
			}

			// Process line breaks
			softLineBreak := false
			if strings.HasSuffix(content, "\n") {
				content = strings.TrimSuffix(content, "\n")
				softLineBreak = true
			}

			fragments := []*Fragment{{
				Value:         content,
				Bold:          bold,
				Italic:        italic,
				Code:          code,
				Link:          link,
				SoftLineBreak: softLineBreak,
			}}

			for _, fragment := range fragments {
				// Only add non-empty fragments
				if fragment.Value != "" {
					currentParagraph.Fragments = append(currentParagraph.Fragments, fragment)
				}
			}
		}
	}

	// Add the last paragraph
	if currentParagraph != nil && len(currentParagraph.Fragments) > 0 {
		body.Paragraphs = append(body.Paragraphs, currentParagraph)
	}

	return body
}
