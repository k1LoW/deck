package deck

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"hash/crc32"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/corona10/goimagehash"
)

type Slides []*Slide

type Slide struct {
	Layout      string   `json:"layout"`
	Freeze      bool     `json:"freeze,omitempty"`
	Titles      []string `json:"titles,omitempty"`
	Subtitles   []string `json:"subtitles,omitempty"`
	Bodies      []*Body  `json:"bodies,omitempty"`
	Images      []*Image `json:"images,omitempty"`
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

type MIMEType string

const (
	MIMETypeImagePNG  MIMEType = "image/png"
	MIMETypeImageJPEG MIMEType = "image/jpeg"
	MIMETypeImageGIF  MIMEType = "image/gif"
)

type Image struct {
	i            image.Image
	mimeType     MIMEType
	url          string // URL if the image was fetched from a URL
	fromMarkdown bool
	checksum     uint32                 // Checksum for the image data
	pHash        *goimagehash.ImageHash // Perceptual hash for JPEG images
	modTime      time.Time              // Modification time of the image file, if applicable
}

func NewImage(pathOrURL string) (*Image, error) {
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

		res, err := client.Get(pathOrURL)
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

func NewImageFromMarkdown(pathOrURL string) (*Image, error) {
	i, err := NewImage(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from code block: %w", err)
	}
	i.fromMarkdown = true
	return i, nil
}

func NewImageFromMarkdownBuffer(buf io.Reader) (*Image, error) {
	i, err := newImageFromBuffer(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from code block: %w", err)
	}
	i.fromMarkdown = true
	return i, nil
}

func newImageFromBuffer(buf io.Reader) (*Image, error) {
	img, mimeType, err := image.Decode(buf)
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
		i:        img,
		mimeType: mt,
	}, nil
}

func (i *Image) Compare(ii *Image) bool {
	if i == nil || ii == nil {
		return false
	}
	if i.mimeType != ii.mimeType {
		return false
	}
	if i.i.Bounds().String() != ii.i.Bounds().String() {
		return false
	}
	if i.Checksum() == i.Checksum() {
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
		i.checksum = crc32.ChecksumIEEE(i.Bytes())
	}
	return i.checksum
}

func (i *Image) PHash() (*goimagehash.ImageHash, error) {
	if i == nil {
		return nil, fmt.Errorf("image is nil")
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
	var buf bytes.Buffer
	switch i.mimeType {
	case MIMETypeImagePNG:
		if err := png.Encode(&buf, i.i); err != nil {
			return ""
		}
	case MIMETypeImageJPEG:
		if err := jpeg.Encode(&buf, i.i, &jpeg.Options{Quality: 100}); err != nil {
			return ""
		}
	case MIMETypeImageGIF:
		if err := gif.Encode(&buf, i.i, nil); err != nil {
			return ""
		}
	default:
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:%s;base64,%s", i.mimeType, encoded)
}

func (i *Image) Bytes() []byte {
	if i == nil {
		return nil
	}
	var buf bytes.Buffer
	switch i.mimeType {
	case MIMETypeImagePNG:
		if err := png.Encode(&buf, i.i); err != nil {
			return nil
		}
	case MIMETypeImageJPEG:
		if err := jpeg.Encode(&buf, i.i, &jpeg.Options{Quality: 100}); err != nil {
			return nil
		}
	case MIMETypeImageGIF:
		if err := gif.Encode(&buf, i.i, nil); err != nil {
			return nil
		}
	default:
		return nil
	}
	return buf.Bytes()
}

func (i *Image) MarshalJSON() ([]byte, error) {
	return []byte(`"` + i.String() + `"`), nil
}

func (i *Image) UnmarshalJSON(data []byte) error {
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
		fmt.Println(string(splitted[1]))
		return fmt.Errorf("failed to decode base64 image data: %w", err)
	}
	img, mimeType, err := image.Decode(bytes.NewReader(decoded))
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}
	if string(i.mimeType) != fmt.Sprintf("image/%s", mimeType) {
		return fmt.Errorf("image MIME type mismatch: expected %s, got %s", i.mimeType, mimeType)
	}
	i.i = img
	return nil
}
