package deck

import "testing"

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
