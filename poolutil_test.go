package deck

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"sync"
	"testing"

	"golang.org/x/sync/errgroup"
)

const (
	basePresentationID = "1wIik04tlp1U4SBHTLrSu20dPFlAGTbRHxnqdRFF9nPo"
	titleForTest       = "For deck integration test (Unless you are testing the deck, you can delete this file without any problems)"
)

// BuildTestOptions creates test options with logger if DECK_TEST_DEBUG is set.
func BuildTestOptions() []Option {
	var opts []Option

	if os.Getenv("DECK_TEST_DEBUG") != "" {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		opts = append(opts, WithLogger(logger))
	}

	if testFolderID := os.Getenv("DECK_TEST_FOLDER_ID"); testFolderID != "" {
		opts = append(opts, WithFolderID(testFolderID))
	}

	return opts
}

// global presentation pool instance.
var presentationPool chan string

// initPresentationPool creates a pool of presentations for parallel tests.
func initPresentationPool(ctx context.Context) ([]string, error) {
	// After trying several times, we decided that 2 parallel is the best setting.
	const parallelCount = 2

	presentationPool = make(chan string, parallelCount)

	// Track created presentations for cleanup
	var created []string
	var mu sync.Mutex

	eg, egCtx := errgroup.WithContext(ctx)

	for i := range parallelCount {
		eg.Go(func() error {
			opts := BuildTestOptions()
			d, err := CreateFrom(egCtx, basePresentationID, opts...)
			if err != nil {
				return fmt.Errorf("failed to create presentation %d: %w", i, err)
			}

			title := fmt.Sprintf("%s (%d)", titleForTest, i)
			if err := d.UpdateTitle(egCtx, title); err != nil {
				return fmt.Errorf("failed to update title for presentation %d: %w", i, err)
			}

			presentationID := d.ID()
			// For snapshot tests with chromedp, we need to make the presentation readable by anyone.
			if err := d.AllowReadingByAnyone(egCtx, presentationID); err != nil {
				return fmt.Errorf("failed to allow reading for presentation %d: %w", i, err)
			}
			mu.Lock()
			created = append(created, presentationID)
			mu.Unlock()

			presentationPool <- presentationID
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return created, nil
}

// TestMain runs setup and cleanup for integration tests.
func TestMain(m *testing.M) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		m.Run()
		return
	}

	// Setup presentation pool
	ctx := context.Background()
	createdPresentations, err := initPresentationPool(ctx)
	if err != nil {
		log.Printf("Failed to setup presentation pool: %v\n", err)
		os.Exit(1)
	}

	m.Run()

	// Cleanup presentations after all tests
	var wg sync.WaitGroup
	for _, id := range createdPresentations {
		wg.Add(1)
		go func(presentationID string) {
			defer wg.Done()

			opts := BuildTestOptions()
			if err := Delete(context.Background(), presentationID, opts...); err != nil {
				log.Printf("Failed to delete presentation %s: %v\n", presentationID, err)
			}
		}(id)
	}
	wg.Wait()
}

// AcquirePresentation gets a presentation ID from the pool.
func AcquirePresentation(t *testing.T) string {
	t.Helper()

	id := <-presentationPool
	t.Cleanup(func() {
		presentationPool <- id
	})
	return id
}
