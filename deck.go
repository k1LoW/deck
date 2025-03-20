package deck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/k1LoW/deck/md"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
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
	bullet md.Bullet
	start  int
	end    int
}

type Slide struct {
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
	if err := d.refresh(); err != nil {
		return nil, err
	}
	return d, nil
}

// Create Google Slides presentation.
func Create(ctx context.Context) (*Deck, error) {
	d, err := initialize(context.Background())
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
	if err := d.refresh(); err != nil {
		return nil, err
	}
	return d, nil
}

// CreateFrom creates a new Deck from the presentation ID.
func CreateFrom(ctx context.Context, id string) (*Deck, error) {
	d, err := initialize(context.Background())
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
	if err := d.refresh(); err != nil {
		return nil, err
	}
	// delete all slides
	if err := d.DeletePageAfter(-1); err != nil {
		return nil, err
	}
	// create first slide
	if err := d.CreatePage(0, &md.Page{
		Layout: d.defaultTitleLayout,
	}); err != nil {
		return nil, err
	}
	return d, nil
}

// List Google Slides presentations.
func List(ctx context.Context) ([]*Slide, error) {
	d, err := initialize(context.Background())
	if err != nil {
		return nil, err
	}
	var slides []*Slide

	r, err := d.driveSrv.Files.List().Q("mimeType='application/vnd.google-apps.presentation'").Fields("files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	for _, f := range r.Files {
		slides = append(slides, &Slide{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	return slides, nil
}

func initialize(ctx context.Context) (*Deck, error) {
	d := &Deck{
		logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
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
func (d *Deck) List() ([]*Slide, error) {
	var slides []*Slide

	r, err := d.driveSrv.Files.List().Q("mimeType='application/vnd.google-apps.presentation'").Fields("files(id, name)").Do()
	if err != nil {
		return nil, err
	}

	for _, f := range r.Files {
		slides = append(slides, &Slide{
			ID:    f.Id,
			Title: f.Name,
		})
	}

	return slides, nil
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
func (d *Deck) Apply(slides md.Slides) error {
	for i, page := range slides {
		if page.Layout == "" {
			switch {
			case i == 0:
				page.Layout = d.defaultTitleLayout
			case len(page.Bodies) == 0:
				page.Layout = d.defaultSectionLayout
			default:
				page.Layout = d.defaultLayout
			}
		}
		if err := d.applyPage(i, page); err != nil {
			return err
		}
	}

	if err := d.DeletePageAfter(len(slides) - 1); err != nil {
		return err
	}

	return nil
}

// ApplyPages applies the markdown slides to the presentation with the specified pages.
func (d *Deck) ApplyPages(slides md.Slides, pages []int) error {
	for i, page := range slides {
		if !slices.Contains(pages, i+1) {
			continue
		}
		if page.Layout == "" {
			switch {
			case i == 0:
				page.Layout = d.defaultTitleLayout
			case len(page.Bodies) == 0:
				page.Layout = d.defaultSectionLayout
			default:
				page.Layout = d.defaultLayout
			}
		}
		if err := d.applyPage(i, page); err != nil {
			return err
		}
	}

	if err := d.DeletePageAfter(len(slides) - 1); err != nil {
		return err
	}

	return nil
}

// UpdateTitle updates the title of the presentation.
func (d *Deck) UpdateTitle(title string) error {
	file := &drive.File{
		Name: title,
	}
	if _, err := d.driveSrv.Files.Update(d.id, file).Do(); err != nil {
		return err
	}
	return nil
}

// Export the presentation as PDF.
func (d *Deck) Export(w io.Writer) error {
	req, err := d.driveSrv.Files.Export(d.id, "application/pdf").Download()
	if err != nil {
		return err
	}
	if err := req.Write(w); err != nil {
		log.Fatalf("Unable to write PDF file: %v", err)
	}
	return nil
}

func (d *Deck) applyPage(index int, page *md.Page) error {
	d.logger.Info("appling page", slog.Int("index", index))
	layoutMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutMap[l.LayoutProperties.DisplayName] = l
	}

	layout, ok := layoutMap[page.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %s", page.Layout)
	}

	if len(d.presentation.Slides) <= index {
		if err := d.CreatePage(index, page); err != nil {
			return err
		}
	}
	if page.Freeze {
		d.logger.Info("skip applying page. because freeze:true", slog.Int("index", index))
		return nil
	}
	currentSlide := d.presentation.Slides[index]
	if currentSlide.SlideProperties.LayoutObjectId != layout.ObjectId {
		// create new page
		if err := d.CreatePage(index+1, page); err != nil {
			return err
		}
		if err := d.DeletePage(index); err != nil {
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
			case "CENTERED_TITLE":
				titles = append(titles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(element.ObjectId); err != nil {
					return err
				}
			case "SUBTITLE":
				subtitles = append(subtitles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(element.ObjectId); err != nil {
					return err
				}
			case "BODY":
				bodies = append(bodies, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				if err := d.clearPlaceholder(element.ObjectId); err != nil {
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
	for i, title := range page.Titles {
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
	for i, subtitle := range page.Subtitles {
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

	// set speacker notes
	req.Requests = append(req.Requests, &slides.Request{
		InsertText: &slides.InsertTextRequest{
			ObjectId: speakerNotesID,
			Text:     strings.Join(page.Comments, "\n\n"),
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
	for i, body := range page.Bodies {
		if len(bodies) <= i {
			continue
		}
		count := 0
		text := ""
		bulletStartIndex = 0 // reset per body
		bulletEndIndex = 0   // reset per body
		var styleReqs []*slides.Request
		currentBullet := md.BulletNone
		for j, paragraph := range body.Paragraphs {
			plen := 0
			if paragraph.Bullet != md.BulletNone {
				if paragraph.Nesting > 0 {
					text += "\t"
					plen++
				}
			}
			for _, fragment := range paragraph.Fragments {
				flen := utf8.RuneCountInString(fragment.Value)
				if fragment.Bold || fragment.Italic {
					var fields []string
					if fragment.Bold {
						fields = append(fields, "bold")
					}
					if fragment.Italic {
						fields = append(fields, "italic")
					}
					styleReqs = append(styleReqs, &slides.Request{
						UpdateTextStyle: &slides.UpdateTextStyleRequest{
							ObjectId: bodies[i].objectID,
							Style: &slides.TextStyle{
								Bold:   fragment.Bold,
								Italic: fragment.Italic,
							},
							TextRange: &slides.Range{
								Type:       "FIXED_RANGE",
								StartIndex: ptrInt64(int64(count + plen)),
								EndIndex:   ptrInt64(int64(count + plen + flen)),
							},
							Fields: strings.Join(fields, ","),
						},
					})
				}
				if fragment.Link != "" {
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
								StartIndex: ptrInt64(int64(count + plen)),
								EndIndex:   ptrInt64(int64(count + plen + flen)),
							},
							Fields: "link",
						},
					})
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
				if paragraph.Bullet != nextParagraph.Bullet || paragraph.Bullet != md.BulletNone {
					text += "\n"
					plen++
				}
			}

			if paragraph.Bullet != md.BulletNone {
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
			req.Requests = append(req.Requests, &slides.Request{
				CreateParagraphBullets: &slides.CreateParagraphBulletsRequest{
					ObjectId:     bodies[i].objectID,
					BulletPreset: convertBullet(r.bullet),
					TextRange: &slides.Range{
						Type:       "FIXED_RANGE",
						StartIndex: ptrInt64(int64(r.start)),
						EndIndex:   ptrInt64(int64(r.end - 1)),
					},
				},
			})
		}
	}
	if len(req.Requests) > 0 {
		if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Do(); err != nil {
			return err
		}
	}

	d.logger.Info("applied page", slog.Int("index", index))
	return nil
}

func (d *Deck) CreatePage(index int, page *md.Page) error {
	layoutMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutMap[l.LayoutProperties.DisplayName] = l
	}

	layout, ok := layoutMap[page.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %s", page.Layout)
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

	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Do(); err != nil {
		return err
	}

	if err := d.refresh(); err != nil {
		return err
	}

	return nil
}

func (d *Deck) DeletePage(index int) error {
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
	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Do(); err != nil {
		return err
	}
	if err := d.refresh(); err != nil {
		return err
	}
	return nil
}

func (d *Deck) DeletePageAfter(index int) error {
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
	if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Do(); err != nil {
		return err
	}
	if err := d.refresh(); err != nil {
		return err
	}
	return nil
}

func (d *Deck) refresh() error {
	presentation, err := d.srv.Presentations.Get(d.id).Do()
	if err != nil {
		return err
	}
	d.presentation = presentation

	// set default layouts
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
	}

	return nil
}

func (d *Deck) clearPlaceholder(placeholderID string) error {
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
					Fields: "bold,italic",
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

	_, _ = d.srv.Presentations.BatchUpdate(d.id, req).Do()
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
	var (
		authCode string
	)
	listenCtx, listening := context.WithCancel(ctx)
	doneCtx, done := context.WithCancel(ctx)
	// run and stop local server
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		authCode = r.URL.Query().Get("code")
		w.Write([]byte("Received code. You may now close this tab."))
		done()
	})
	srv := &http.Server{Handler: handler}
	var listenErr error
	go func() {
		ln, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			listenErr = fmt.Errorf("Listen: %w", err)
			listening()
			done()
			return
		}
		srv.Addr = ln.Addr().String()
		listening()
		if err := srv.Serve(ln); err != nil {
			if err != http.ErrServerClosed {
				log.Fatalf("ListenAndServe: %v", err)
			}
		}
	}()
	<-listenCtx.Done()
	if listenErr != nil {
		return nil, listenErr
	}
	config.RedirectURL = "http://" + srv.Addr + "/"

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	if err := open.Start(authURL); err != nil {
		return nil, err
	}

	<-doneCtx.Done()
	if err := srv.Shutdown(ctx); err != nil {
		return nil, err
	}

	token, err := config.Exchange(ctx, authCode)
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
		return fmt.Errorf("Unable to cache oauth token: %w", err)
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("Unable to cache oauth token: %w", err)
	}
	return nil
}

func ptrInt64(i int64) *int64 {
	return &i
}

func convertBullet(b md.Bullet) string {
	switch b {
	case md.BulletDash:
		return "BULLET_DISC_CIRCLE_SQUARE"
	case md.BulletNumber:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	case md.BulletAlpha:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	default:
		return "UNRECOGNIZED"
	}
}
