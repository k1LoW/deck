package deck

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// currentImageData holds the result of parallel image fetching
type currentImageData struct {
	currentImages           []*Image
	currentImageObjectIDMap map[*Image]string
}

// imageToPreload holds image information with slide context
type imageToPreload struct {
	slideIndex     int
	imageIndex     int    // index within the slide
	existingURL    string // URL of existing image
	objectID       string // objectID of existing image
	isFromMarkdown bool   // whether this image is from markdown
}

// imageResult holds the result of image processing
type imageResult struct {
	slideIndex int
	imageIndex int
	image      *Image
	objectID   string
}

// preloadCurrentImages pre-fetches current images for all slides that will be processed
func (d *Deck) preloadCurrentImages(ctx context.Context, actions []*action) (map[int]*currentImageData, error) {
	result := make(map[int]*currentImageData)

	// Collect all images that need preloading
	var imagesToPreload []imageToPreload

	for _, action := range actions {
		switch action.actionType {
		case actionTypeUpdate:
			// Extract existing images from the current slide
			if action.index < len(d.presentation.Slides) {
				currentSlide := d.presentation.Slides[action.index]
				imageIndexInSlide := 0
				for _, element := range currentSlide.PageElements {
					if element.Image != nil && element.Image.Placeholder == nil && element.Image.ContentUrl != "" {
						imagesToPreload = append(imagesToPreload, imageToPreload{
							slideIndex:     action.index,
							imageIndex:     imageIndexInSlide,
							existingURL:    element.Image.ContentUrl,
							objectID:       element.ObjectId,
							isFromMarkdown: element.Description == descriptionImageFromMarkdown,
						})
						imageIndexInSlide++
					}
				}
			}
		}
	}

	if len(imagesToPreload) == 0 {
		return result, nil
	}

	// Process images in parallel
	const maxWorkers = 8

	imageCh := make(chan imageToPreload, len(imagesToPreload))
	resultCh := make(chan imageResult, len(imagesToPreload))

	// Start worker goroutines
	g, ctx := errgroup.WithContext(ctx)
	numWorkers := min(maxWorkers, len(imagesToPreload))

	for range numWorkers {
		g.Go(func() error {
			for imgToPreload := range imageCh {
				var image *Image
				var err error

				// Create Image from existing URL
				if imgToPreload.isFromMarkdown {
					image, err = NewImageFromMarkdown(imgToPreload.existingURL)
				} else {
					image, err = NewImage(imgToPreload.existingURL)
				}
				if err != nil {
					return fmt.Errorf("failed to preload image from URL %s: %w", imgToPreload.existingURL, err)
				}

				select {
				case resultCh <- imageResult{
					slideIndex: imgToPreload.slideIndex,
					imageIndex: imgToPreload.imageIndex,
					image:      image,
					objectID:   imgToPreload.objectID,
				}:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			return nil
		})
	}

	// Send work to workers
	go func() {
		defer close(imageCh)
		for _, imgToPreload := range imagesToPreload {
			imageCh <- imgToPreload
		}
	}()

	// Close result channel when all workers are done
	go func() {
		g.Wait()
		close(resultCh)
	}()

	// Collect results and build currentImageData directly with proper ordering
	for res := range resultCh {
		if res.image != nil {
			if result[res.slideIndex] == nil {
				result[res.slideIndex] = &currentImageData{
					currentImages:           []*Image{},
					currentImageObjectIDMap: map[*Image]string{},
				}
			}

			// Resize currentImages slice if needed
			if len(result[res.slideIndex].currentImages) <= res.imageIndex {
				newSize := res.imageIndex + 1
				newSlice := make([]*Image, newSize)
				copy(newSlice, result[res.slideIndex].currentImages)
				result[res.slideIndex].currentImages = newSlice
			}

			// Place image at the correct index
			result[res.slideIndex].currentImages[res.imageIndex] = res.image
			result[res.slideIndex].currentImageObjectIDMap[res.image] = res.objectID
		}
	}

	// Wait for all workers to complete and check for errors
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to preload images: %w", err)
	}

	return result, nil
}
