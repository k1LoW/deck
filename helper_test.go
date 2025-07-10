package deck

import (
	"context"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

const (
	viewportWidth  = 1280
	viewportHeight = 800
)

func Screenshot(t *testing.T, url string) []byte {
	t.Helper()
	ctx, cancel := chromedp.NewContext(
		context.Background(),
	)
	t.Cleanup(func() {
		cancel()
	})
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
	})
	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(viewportWidth, viewportHeight),
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.FullScreenshot(&buf, 100),
	); err != nil {
		t.Fatalf("Failed to take screenshot: %v", err)
	}

	return buf
}
