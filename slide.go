package deck

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/k1LoW/errors"
)

type Slides []*Slide

type Slide struct {
	Layout         string        `json:"layout"`
	Freeze         bool          `json:"freeze,omitempty"`
	Skip           bool          `json:"skip,omitempty"`
	Titles         []string      `json:"titles,omitempty"`
	TitleBodies    []*Body       `json:"title_bodies,omitempty"`
	Subtitles      []string      `json:"subtitles,omitempty"`
	SubtitleBodies []*Body       `json:"subtitle_bodies,omitempty"`
	Bodies         []*Body       `json:"bodies,omitempty"`
	Images         []*Image      `json:"images,omitempty"`
	BlockQuotes    []*BlockQuote `json:"block_quotes,omitempty"`
	Tables         []*Table      `json:"tables,omitempty"`
	SpeakerNote    string        `json:"speaker_note,omitempty"`

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
	Value     string `json:"value"`
	Bold      bool   `json:"bold,omitempty"`
	Italic    bool   `json:"italic,omitempty"`
	Link      string `json:"link,omitempty"`
	Code      bool   `json:"code,omitempty"`
	StyleName string `json:"style_name,omitempty"`
}

type BlockQuote struct {
	Paragraphs []*Paragraph `json:"paragraphs,omitempty"`
	Nesting    int          `json:"nesting,omitempty"`
}

type Table struct {
	Rows []*TableRow `json:"rows,omitempty"`
}

type TableRow struct {
	Cells []*TableCell `json:"cells,omitempty"`
}

type TableCell struct {
	Fragments []*Fragment `json:"content,omitempty"`
	Alignment string      `json:"alignment,omitempty"`
	IsHeader  bool        `json:"is_header,omitempty"`
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

type MIMEType string

const (
	MIMETypeImagePNG  MIMEType = "image/png"
	MIMETypeImageJPEG MIMEType = "image/jpeg"
	MIMETypeImageGIF  MIMEType = "image/gif"
)

type Image struct {
	i            image.Image
	b            []byte // Raw image data
	mimeType     MIMEType
	url          string // URL if the image was fetched from a URL
	fromMarkdown bool
	checksum     uint32                 // Checksum for the image data
	pHash        *goimagehash.ImageHash // Perceptual hash for JPEG images
	modTime      time.Time              // Modification time of the image file, if applicable
	codeBlock    bool                   // Whether the image was created from a code block

	// Upload state management
	uploadMutex    sync.RWMutex
	uploadState    uploadState
	webContentLink string
	uploadError    error
}

type uploadState int

const (
	uploadStateNotStarted uploadState = iota
	uploadStateInProgress
	uploadStateCompleted
	uploadStateFailed
)

func (b *Body) String() string {
	var result strings.Builder
	for i, paragraph := range b.Paragraphs {
		if i > 0 && b.Paragraphs[i-1].Bullet != BulletNone && paragraph.Bullet == BulletNone {
			result.WriteString("\n")
		}
		result.WriteString(paragraph.String())
		switch {
		case paragraph.Bullet != BulletNone:
			result.WriteString("\n")
		case i == len(b.Paragraphs)-1:
			result.WriteString("\n")
		default:
			result.WriteString("\n\n")
		}
	}
	return result.String()
}

func (p *Paragraph) String() string {
	if p == nil {
		return ""
	}
	var result strings.Builder
	result.WriteString(strings.Repeat("  ", p.Nesting))
	switch p.Bullet {
	case BulletDash:
		result.WriteString("- ")
	case BulletNumber:
		result.WriteString("1. ")
	case BulletAlpha:
		result.WriteString("a. ")
	}
	for _, fragment := range p.Fragments {
		if fragment == nil {
			continue
		}
		result.WriteString(fragment.Value)
	}
	return result.String()
}

func (b *BlockQuote) String() string {
	if b == nil {
		return ""
	}
	quotes := strings.Repeat("> ", b.Nesting+1)
	var result strings.Builder
	for i, paragraph := range b.Paragraphs {
		result.WriteString(quotes)
		if i > 0 && b.Paragraphs[i-1].Bullet != BulletNone && paragraph.Bullet == BulletNone {
			result.WriteString("\n")
			result.WriteString(quotes)
		}
		result.WriteString(paragraph.String())
		switch {
		case paragraph.Bullet != BulletNone:
			result.WriteString("\n")
		case i == len(b.Paragraphs)-1:
			result.WriteString("\n")
		default:
			result.WriteString("\n")
			result.WriteString(quotes)
			result.WriteString("\n")
		}
	}
	return result.String()
}

func NewImage(pathOrURL string) (_ *Image, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	var b io.Reader
	var modTime time.Time
	if strings.HasPrefix(pathOrURL, "http://") || strings.HasPrefix(pathOrURL, "https://") {
		i, ok := LoadImageCache(pathOrURL)
		if ok {
			return i, nil
		}
		if _, err := url.Parse(pathOrURL); err != nil {
			return nil, fmt.Errorf("invalid URL %s: %w", pathOrURL, err)
		}

		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		req, err := http.NewRequest("GET", pathOrURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch image from URL %s: %w", pathOrURL, err)
		}
		req.Header.Set("User-Agent", userAgent)
		res, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch image from URL %s: %w", pathOrURL, err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch image from URL %s: status code %d", pathOrURL, res.StatusCode)
		}
		b = res.Body
	} else {
		fi, err := os.Stat(pathOrURL)
		if err != nil {
			return nil, fmt.Errorf("failed to stat image file %s: %w", pathOrURL, err)
		}
		modTime = fi.ModTime()
		i, ok := LoadImageCache(pathOrURL)
		if ok {
			if modTime.Equal(i.modTime) {
				return i, nil
			}
		}
		file, err := os.Open(pathOrURL)
		if err != nil {
			return nil, fmt.Errorf("failed to open image file %s: %w", pathOrURL, err)
		}
		defer file.Close()
		b = file
	}
	i, err := newImageFromBuffer(b)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from buffer: %w", err)
	}
	i.url = pathOrURL
	i.modTime = modTime
	StoreImageCache(pathOrURL, i)
	return i, nil
}

func NewImageFromMarkdown(pathOrURL string) (_ *Image, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	i, err := NewImage(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from path or URL: %w", err)
	}
	i.fromMarkdown = true
	return i, nil
}

func NewImageFromCodeBlock(r io.Reader) (_ *Image, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	i, err := newImageFromBuffer(r)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from code block: %w", err)
	}
	i.fromMarkdown = true
	i.codeBlock = true
	return i, nil
}

func newImageFromBuffer(r io.Reader) (_ *Image, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}
	_, mimeType, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	var mt MIMEType
	switch mimeType {
	case "png":
		mt = MIMETypeImagePNG
	case "jpeg":
		mt = MIMETypeImageJPEG
	case "gif":
		mt = MIMETypeImageGIF
	default:
		return nil, fmt.Errorf("unsupported image MIME type: %s", mimeType)
	}
	return &Image{
		b:        b,
		mimeType: mt,
	}, nil
}

func (i *Image) Equivalent(ii *Image) bool {
	if i == nil || ii == nil {
		return false
	}
	if i.mimeType != ii.mimeType {
		return false
	}
	if i.Checksum() == ii.Checksum() {
		return true
	}
	if i.mimeType == MIMETypeImageJPEG {
		// Only JPEG images are compressed on the Google Slides side,
		// so we use Perceptual Hashing for comparison
		aHash, err := i.PHash()
		if err != nil {
			return false
		}
		bHash, err := ii.PHash()
		if err != nil {
			return false
		}
		distance, err := aHash.Distance(bHash)
		if err != nil {
			return false
		}
		if distance < 5 { // threshold for similarity
			return true
		}
	}
	return false
}

func (i *Image) Checksum() uint32 {
	if i == nil {
		return 0
	}
	if i.checksum == 0 {
		i.checksum = crc32.ChecksumIEEE(i.b)
	}
	return i.checksum
}

func (i *Image) Image() (image.Image, error) {
	if i == nil {
		return nil, fmt.Errorf("image is nil")
	}
	if i.i == nil {
		img, _, err := image.Decode(bytes.NewReader(i.b))
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
		i.i = img
	}
	return i.i, nil
}

func (i *Image) PHash() (_ *goimagehash.ImageHash, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if i == nil {
		return nil, fmt.Errorf("image is nil")
	}
	if i.i == nil {
		if _, err := i.Image(); err != nil {
			return nil, err
		}
	}
	if i.pHash == nil {
		pHash, err := goimagehash.PerceptionHash(i.i)
		if err != nil {
			return nil, fmt.Errorf("failed to compute perceptual hash: %w", err)
		}
		i.pHash = pHash
	}
	return i.pHash, nil
}

func (i *Image) String() string {
	if i == nil {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(i.b)
	return fmt.Sprintf("data:%s;base64,%s", i.mimeType, encoded)
}

func (i *Image) Bytes() []byte {
	if i == nil {
		return nil
	}
	return i.b
}

func (i *Image) MarshalJSON() (_ []byte, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	return []byte(`"` + i.String() + `"`), nil
}

func (i *Image) UnmarshalJSON(data []byte) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	data = bytes.Trim(data, `"`)
	if !bytes.HasPrefix(data, []byte(`data:`)) {
		return fmt.Errorf("invalid image data: %s", data)
	}
	splitted := bytes.Split(bytes.TrimPrefix(data, []byte(`data:`)), []byte(";base64,"))
	if len(splitted) != 2 {
		return fmt.Errorf("invalid image data: %s", data)
	}
	i.mimeType = MIMEType(splitted[0])
	decoded, err := base64.StdEncoding.DecodeString(string(splitted[1]))
	if err != nil {
		return fmt.Errorf("failed to decode base64 image data: %w", err)
	}
	_, mimeType, err := image.DecodeConfig(bytes.NewReader(decoded))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	if string(i.mimeType) != fmt.Sprintf("image/%s", mimeType) {
		return fmt.Errorf("image MIME type mismatch: expected %s, got %s", i.mimeType, mimeType)
	}
	i.b = decoded
	return nil
}

// StartUpload marks the image as upload in progress.
func (i *Image) StartUpload() {
	i.uploadMutex.Lock()
	defer i.uploadMutex.Unlock()
	i.uploadState = uploadStateInProgress
}

// SetUploadResult sets the upload result (success or failure).
func (i *Image) SetUploadResult(webContentLink string, err error) {
	i.uploadMutex.Lock()
	defer i.uploadMutex.Unlock()
	if err != nil {
		i.uploadState = uploadStateFailed
		i.uploadError = err
	} else {
		i.uploadState = uploadStateCompleted
		i.webContentLink = webContentLink
		i.uploadError = nil
	}
}

type uploadInfo struct {
	url       string
	codeBlock bool
}

// UploadInfo waits for the upload to complete and returns the webContentLink.
func (i *Image) UploadInfo(ctx context.Context) (*uploadInfo, error) {
	for {
		i.uploadMutex.RLock()
		state := i.uploadState
		link := i.webContentLink
		uploadErr := i.uploadError
		i.uploadMutex.RUnlock()

		switch state {
		case uploadStateNotStarted:
			// Image upload not started, return empty value
			return nil, nil
		case uploadStateCompleted:
			return &uploadInfo{
				url:       link,
				codeBlock: i.codeBlock,
			}, nil
		case uploadStateFailed:
			return nil, uploadErr
		case uploadStateInProgress:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Millisecond):
				// Continue waiting
			}
		}
	}
}

// IsUploadNeeded returns true if the image needs to be uploaded.
func (i *Image) IsUploadNeeded() bool {
	i.uploadMutex.RLock()
	defer i.uploadMutex.RUnlock()
	return i.uploadState == uploadStateNotStarted
}

func (i *Image) ClearUploadState() {
	i.uploadMutex.Lock()
	defer i.uploadMutex.Unlock()
	i.uploadState = uploadStateNotStarted
	i.webContentLink = ""
	i.uploadError = nil
}
