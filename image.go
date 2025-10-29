package deck

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/corona10/goimagehash"
	"github.com/k1LoW/errors"
	"golang.org/x/net/publicsuffix"
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
	link         string                 // External link associated with the image

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
	if isPublicURL(pathOrURL) {
		// If the URL appears to be OK for direct access, `deck` will not upload a temporary image to Google Drive
		// but will instead specify that URL directly in the CreateImageRequest.
		i.webContentLink = pathOrURL
	}
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

func (i *Image) SetLink(link string) {
	i.link = link
}

func (i *Image) Equivalent(ii *Image) bool {
	if i == nil || ii == nil {
		return false
	}
	if i.mimeType != ii.mimeType {
		return false
	}
	if i.link != ii.link {
		return false
	}
	if i.Checksum() == ii.Checksum() {
		return true
	}

	// Images are compressed on the Google Slides side (especially JPEG, large images),
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

// internalImage is a subset of `Image` that excludes state and other elements, containing the minimum
// data required to reproduce the `Image`. It is used for `json.Marshal/Unmarshal` and caching purposes.
type internalImage struct {
	Data         string
	URL          string
	FromMarkdown bool
	ModTime      time.Time
	Link         string
}

// MarshalJSON and UnmarshalJSON are defined for cloning data and for similarity comparisons of `slide` structures.
func (i *Image) MarshalJSON() (_ []byte, err error) {
	return json.Marshal(i.toInternal())
}

func (i *Image) UnmarshalJSON(data []byte) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	var iimg internalImage
	if err := json.Unmarshal(data, &iimg); err != nil {
		return fmt.Errorf("failed to unmarshal image data: %w", err)
	}
	return iimg.toImage(i)
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
	link      string
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
		case uploadStateNotStarted, uploadStateCompleted:
			return &uploadInfo{
				url:       link,
				link:      i.link,
				codeBlock: i.codeBlock(),
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
	return i.uploadState == uploadStateNotStarted && i.webContentLink == ""
}

func (i *Image) codeBlock() bool {
	return i.url == "" && i.fromMarkdown
}

func (i *Image) toInternal() *internalImage {
	return &internalImage{
		Data:         i.String(),
		URL:          i.url,
		FromMarkdown: i.fromMarkdown,
		ModTime:      i.modTime,
		Link:         i.link,
	}
}

func (iimg *internalImage) toImage(i *Image) error {
	i.url = iimg.URL
	i.fromMarkdown = iimg.FromMarkdown
	i.modTime = iimg.ModTime
	i.link = iimg.Link

	data := []byte(iimg.Data)
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

// isPublicURL checks whether a URL string is OK for direct public access.
// Since we only need to identify what appear to be public URLs, false negatives are acceptable.
func isPublicURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.User != nil || u.Port() != "" {
		return false
	}
	if ip := net.ParseIP(u.Host); ip != nil {
		return false
	}
	_, icann := publicsuffix.PublicSuffix(u.Host)
	return icann
}
