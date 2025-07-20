package deck

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/api/drive/v3"
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

// startUploadingImages starts uploading new images asynchronously
func (d *Deck) startUploadingImages(ctx context.Context, actions []*action, currentImages map[int]*currentImageData) {
	// Collect all images that need uploading
	var imagesToUpload []*Image

	for _, action := range actions {
		switch action.actionType {
		case actionTypeUpdate, actionTypeAppend, actionTypeInsert:
			if action.slide == nil {
				continue
			}
			for _, image := range action.slide.Images {
				// Check if this image already exists in current images
				var found bool
				if currentImagesForSlide, exists := currentImages[action.index]; exists {
					found = slices.ContainsFunc(currentImagesForSlide.currentImages, func(currentImage *Image) bool {
						return currentImage.Compare(image)
					})
				}
				if !found && image.IsUploadNeeded() {
					imagesToUpload = append(imagesToUpload, image)
				}
			}
		}
	}

	if len(imagesToUpload) == 0 {
		return
	}

	// Mark all images as upload in progress
	for _, image := range imagesToUpload {
		image.StartUpload()
	}

	// Start uploading images asynchronously
	go func() {
		// Process images in parallel
		const maxWorkers = 8

		imageCh := make(chan *Image, len(imagesToUpload))

		// Start worker goroutines
		g, uploadCtx := errgroup.WithContext(ctx)
		numWorkers := min(maxWorkers, len(imagesToUpload))

		for range numWorkers {
			g.Go(func() error {
				for image := range imageCh {
					// Upload image to Google Drive
					df := &drive.File{
						Name:     fmt.Sprintf("________tmp-for-deck-%s", time.Now().Format(time.RFC3339)),
						MimeType: string(image.mimeType),
					}
					uploaded, err := d.driveSrv.Files.Create(df).Media(bytes.NewBuffer(image.Bytes())).Do()
					if err != nil {
						image.SetUploadResult("", "", fmt.Errorf("failed to upload image: %w", err))
						continue
					}

					// Set permission
					if _, err := d.driveSrv.Permissions.Create(uploaded.Id, &drive.Permission{
						Type: "anyone",
						Role: "reader",
					}).Do(); err != nil {
						// Clean up uploaded file on permission error
						d.driveSrv.Files.Delete(uploaded.Id).Do()
						image.SetUploadResult("", "", fmt.Errorf("failed to set permission for image: %w", err))
						continue
					}

					// Get webContentLink
					f, err := d.driveSrv.Files.Get(uploaded.Id).Fields("webContentLink").Do()
					if err != nil {
						// Clean up uploaded file on error
						d.driveSrv.Files.Delete(uploaded.Id).Do()
						image.SetUploadResult("", "", fmt.Errorf("failed to get webContentLink for image: %w", err))
						continue
					}

					if f.WebContentLink == "" {
						// Clean up uploaded file on error
						d.driveSrv.Files.Delete(uploaded.Id).Do()
						image.SetUploadResult("", "", fmt.Errorf("webContentLink is empty for image: %s", uploaded.Id))
						continue
					}

					// Set successful upload result
					image.SetUploadResult(f.WebContentLink, uploaded.Id, nil)
				}
				return nil
			})
		}

		// Send work to workers
		go func() {
			defer close(imageCh)
			for _, image := range imagesToUpload {
				select {
				case imageCh <- image:
				case <-uploadCtx.Done():
					return
				}
			}
		}()

		// Wait for all workers to complete
		if err := g.Wait(); err != nil {
			d.logger.Error("failed to upload some images", slog.Any("error", err))
		}
	}()
}
