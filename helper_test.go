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

var browserCtx context.Context

// InitChrome initializes and preloads a Chrome browser context for testing.
func InitChrome(ctx context.Context) func() {
	aCtx, aCancel := chromedp.NewExecAllocator(ctx, chromedp.DefaultExecAllocatorOptions[:]...)
	var bCancel context.CancelFunc
	browserCtx, bCancel = chromedp.NewContext(aCtx)
	return func() {
		bCancel()
		aCancel()
	}
}

func Screenshot(t *testing.T, url string) []byte {
	t.Helper()
	var ctx = browserCtx
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := chromedp.NewContext(ctx)
	t.Cleanup(cancel)

	ctx, cancel2 := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(cancel2)

	var buf []byte
	if err := chromedp.Run(ctx,
		chromedp.EmulateViewport(viewportWidth, viewportHeight),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.FullScreenshot(&buf, 100),
	); err != nil {
		t.Fatalf("Failed to take screenshot: %v", err)
	}

	return buf
}
