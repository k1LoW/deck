package deck

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestIsPulicURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"Valid public URL", "http://example.com/image.jpg", true},
		{"Valid public URL with HTTPS", "https://example.com/image.jpg", true},
		{"Valid public URL but with port", "http://example.com:8080/image.jpg", false},
		{"Valid URL but wity auth", "https://user:passwd@example.com/image.jpg", false},
		{"Invalid URL with invalid suffix", "https://example.invalid/image.jpg", false},
		{"Invalid URL scheme", "ftp://example.com/image.jpg", false},
		{"File URL", "file:///path/to/image.jpg", false},
		{"Localhost URL", "http://localhost/image.jpg", false},
		{"Private IP URL", "http://192.168.0.1/image.jpg", false},
		{"Public IP URL", "http:192.0.2.0/image.jpg", false},
		{"IPv6 URL", "http://[fd00::1]/image.jpg", false},
		{"Malformed URL", "http:///image.jpg", false},
		{"Empty URL", "", false},
		{"Local File Abs Path", "/path/to/image.jpg", false},
		{"Local File Rel Path", "./path/to/image.jpg", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isPublicURL(tt.url); got != tt.want {
				t.Errorf("isPublicURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func dummyPNG(t *testing.T) *bytes.Buffer {
	t.Helper()
	// Create a 1x1 pixel stub PNG image
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255}) // Red pixel
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
	return &buf
}

func TestNewImageFromCodeBlock(t *testing.T) {
	buf := dummyPNG(t)
	i, err := NewImageFromCodeBlock(buf)
	if err != nil {
		t.Fatalf("TestNewImageFromCodeBlock failed: %v", err)
	}
	if got := i.codeBlock(); !got {
		t.Errorf("Image.codeBlock() = %v, want true", got)
	}
}
