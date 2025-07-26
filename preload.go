package deck

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"google.golang.org/api/drive/v3"
)

const maxPreloadWorkersNum = 4

// currentImageData holds the result of parallel image fetching.
type currentImageData struct {
	currentImages           []*Image
	currentImageObjectIDMap map[*Image]string
}

// imageToPreload holds image information with slide context.
type imageToPreload struct {
	slideIndex     int
	imageIndex     int    // index within the slide
	existingURL    string // URL of existing image
	objectID       string // objectID of existing image
	isFromMarkdown bool   // whether this image is from markdown
}

// imageResult holds the result of image processing.
type imageResult struct {
	slideIndex int
	imageIndex int
	image      *Image
	objectID   string
}

// preloadCurrentImages pre-fetches current images for all slides that will be processed.
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
	d.logger.Info("preloading current images", slog.Int("count", len(imagesToPreload)))

	// Process images in parallel
	sem := semaphore.NewWeighted(maxPreloadWorkersNum)
	eg, ctx := errgroup.WithContext(ctx)
	resultCh := make(chan imageResult, len(imagesToPreload))

	for _, imgToPreload := range imagesToPreload {
		eg.Go(func() error {
			// Try to acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

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

			resultCh <- imageResult{
				slideIndex: imgToPreload.slideIndex,
				imageIndex: imgToPreload.imageIndex,
				image:      image,
				objectID:   imgToPreload.objectID,
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("failed to preload images: %w", err)
	}
	close(resultCh)

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

	d.logger.Info("preloaded current images")
	return result, nil
}

// uploadedImageInfo holds information about uploaded images for cleanup.
type uploadedImageInfo struct {
	uploadedID string
	image      *Image
}

// startUploadingImages starts uploading new images asynchronously and returns a channel for cleanup.
func (d *Deck) startUploadingImages(
	ctx context.Context, actions []*action, currentImages map[int]*currentImageData) <-chan uploadedImageInfo {

	// Collect all images that need uploading
	var imagesToUpload []*Image

	for _, action := range actions {
		switch action.actionType {
		case actionTypeUpdate, actionTypeAppend:
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

	// Create channel for uploaded image IDs
	uploadedCh := make(chan uploadedImageInfo, len(imagesToUpload))
	if len(imagesToUpload) == 0 {
		close(uploadedCh)
		return uploadedCh
	}
	d.logger.Info("starting image upload", slog.Int("count", len(imagesToUpload)))

	// Mark all images as upload in progress
	for _, image := range imagesToUpload {
		image.StartUpload()
	}

	// Start uploading images asynchronously
	go func() {
		// Process images in parallel
		sem := semaphore.NewWeighted(maxPreloadWorkersNum)
		eg, ctx := errgroup.WithContext(ctx)

		for _, image := range imagesToUpload {
			eg.Go(func() error {
				if err := sem.Acquire(ctx, 1); err != nil {
					// Context canceled, set upload error on remaining images
					image.SetUploadResult("", "", err)
					return nil
				}
				defer sem.Release(1)

				// Upload image to Google Drive
				df := &drive.File{
					Name:     fmt.Sprintf("________tmp-for-deck-%s", time.Now().Format(time.RFC3339)),
					MimeType: string(image.mimeType),
				}
				uploaded, err := d.driveSrv.Files.Create(df).Media(bytes.NewBuffer(image.Bytes())).Do()
				if err != nil {
					image.SetUploadResult("", "", fmt.Errorf("failed to upload image: %w", err))
					return nil
				}

				// Set permission
				if _, err := d.driveSrv.Permissions.Create(uploaded.Id, &drive.Permission{
					Type: "anyone",
					Role: "reader",
				}).Do(); err != nil {
					// Clean up uploaded file on permission error
					if deleteErr := d.driveSrv.Files.Delete(uploaded.Id).Do(); deleteErr != nil {
						d.logger.Error("failed to delete uploaded file after permission error",
							slog.String("id", uploaded.Id),
							slog.Any("error", deleteErr))
					}
					image.SetUploadResult("", "", fmt.Errorf("failed to set permission for image: %w", err))
					return nil
				}

				// Get webContentLink
				f, err := d.driveSrv.Files.Get(uploaded.Id).Fields("webContentLink").Do()
				if err != nil {
					// Clean up uploaded file on error
					if deleteErr := d.driveSrv.Files.Delete(uploaded.Id).Do(); deleteErr != nil {
						d.logger.Error("failed to delete uploaded file after webContentLink fetch error",
							slog.String("id", uploaded.Id),
							slog.Any("error", deleteErr))
					}
					image.SetUploadResult("", "", fmt.Errorf("failed to get webContentLink for image: %w", err))
					return nil
				}

				if f.WebContentLink == "" {
					// Clean up uploaded file on error
					if deleteErr := d.driveSrv.Files.Delete(uploaded.Id).Do(); deleteErr != nil {
						d.logger.Error("failed to delete uploaded file after empty webContentLink",
							slog.String("id", uploaded.Id),
							slog.Any("error", deleteErr))
					}
					image.SetUploadResult("", "", fmt.Errorf("webContentLink is empty for image: %s", uploaded.Id))
					return nil
				}

				// Set successful upload result
				image.SetUploadResult(f.WebContentLink, uploaded.Id, nil)

				uploadedCh <- uploadedImageInfo{uploadedID: uploaded.Id, image: image}
				return nil
			})
		}

		// Wait for all workers to complete
		if err := eg.Wait(); err != nil {
			d.logger.Error("failed to upload images", slog.Any("error", err))
		}
		// Close the channel when all uploads are done
		close(uploadedCh)
	}()

	return uploadedCh
}

// cleanupUploadedImages deletes uploaded images in parallel.
func (d *Deck) cleanupUploadedImages(ctx context.Context, uploadedCh <-chan uploadedImageInfo) error {
	sem := semaphore.NewWeighted(maxPreloadWorkersNum)
	var wg sync.WaitGroup

	for {
		select {
		case info, ok := <-uploadedCh:
			if !ok {
				// Channel closed, wait for all deletions to complete
				wg.Wait()
				return nil
			}
			// Try to acquire semaphore
			if err := sem.Acquire(ctx, 1); err != nil {
				return fmt.Errorf("failed to acquire semaphore: %w", err)
			}

			wg.Add(1)
			go func(info uploadedImageInfo) {
				defer func() {
					sem.Release(1)
					wg.Done()
				}()

				// Delete uploaded image from Google Drive
				// Note: We only log errors here instead of returning them to ensure
				// all images are attempted to be deleted. A single deletion failure
				// should not prevent cleanup of other successfully uploaded images.
				if err := d.driveSrv.Files.Delete(info.uploadedID).Do(); err != nil {
					d.logger.Error("failed to delete uploaded image",
						slog.String("id", info.uploadedID),
						slog.Any("error", err))
				}
			}(info)

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
