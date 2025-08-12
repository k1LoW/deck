package deck

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/k1LoW/errors"
	"google.golang.org/api/slides/v1"
)

const (
	styleBlockQuote                = "blockquote"
	descriptionImageFromMarkdown   = "Image generated from markdown"
	descriptionTextboxFromMarkdown = "Textbox generated from markdown"
)

// Apply the markdown slides to the presentation.
func (d *Deck) Apply(ctx context.Context, slides Slides) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	pages := make([]int, 0, len(slides))
	for i := range len(slides) {
		pages = append(pages, i+1)
	}
	return d.ApplyPages(ctx, slides, pages)
}

// ApplyPages applies the markdown slides to the presentation with the specified pages.
func (d *Deck) ApplyPages(ctx context.Context, ss Slides, pages []int) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	if slices.ContainsFunc(pages, func(page int) bool {
		return page < 1 || page > len(ss)
	}) {
		return fmt.Errorf("invalid page number in pages: %v", pages)
	}

	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	layoutObjectIdMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutObjectIdMap[l.ObjectId] = l
	}

	before := make(Slides, len(d.presentation.Slides))
	after := make(Slides, len(d.presentation.Slides))
	for i, p := range d.presentation.Slides {
		slide := convertToSlide(p, layoutObjectIdMap)
		before[i] = slide
		after[i] = slide
	}

	for _, page := range pages {
		i := page - 1
		slide := ss[i]
		if slide.Layout == "" {
			if i == 0 {
				slide.Layout = d.defaultTitleLayout
			} else {
				slide.Layout = d.defaultLayout
			}
		}
		if i < len(after) {
			after[i] = slide
		} else {
			after = append(after, slide)
		}
	}
	if len(after) > len(ss) {
		after = after[:len(ss)]
	}

	actions, err := generateActions(before, after)
	if err != nil {
		return fmt.Errorf("failed to generate actions: %w", err)
	}

	var actionDetails []actionDetail
	for _, action := range actions {
		switch action.actionType {
		case actionTypeAppend:
			actionDetails = append(actionDetails, actionDetail{
				ActionType:  actionTypeAppend,
				Titles:      action.slide.Titles,
				Index:       nil,
				MoveToIndex: nil,
			})
		case actionTypeUpdate:
			actionDetails = append(actionDetails, actionDetail{
				ActionType:  actionTypeUpdate,
				Titles:      action.slide.Titles,
				Index:       &action.index,
				MoveToIndex: nil,
			})
		case actionTypeMove:
			actionDetails = append(actionDetails, actionDetail{
				ActionType:  actionTypeMove,
				Titles:      action.slide.Titles,
				Index:       &action.index,
				MoveToIndex: &action.moveToIndex,
			})
		}
	}

	// Pre-fetch current images in parallel for only the slides that will be updated
	currentImages, err := d.preloadCurrentImages(ctx, actions)
	if err != nil {
		return fmt.Errorf("failed to preload current images: %w", err)
	}

	// Start uploading new images in parallel (don't wait for completion)
	uploadedCh := d.startUploadingImages(ctx, actions, currentImages)
	defer func() {
		// Clean up uploaded images in parallel
		if cleanupErr := d.cleanupUploadedImages(ctx, uploadedCh); cleanupErr != nil {
			if err == nil {
				err = fmt.Errorf("failed to cleanup uploaded images: %w", cleanupErr)
			} else {
				d.logger.Error("failed to cleanup uploaded images", slog.Any("error", cleanupErr))
			}
		}
		ClearAllUploadStateFromCache()
	}()

	d.logger.Info("applying actions", slog.Any("actions", actionDetails))

	var layoutsForAppendPages []string
	for _, action := range actions {
		if action.actionType == actionTypeAppend {
			layoutsForAppendPages = append(layoutsForAppendPages, action.slide.Layout)
		}
	}

	currentSlidesLen := len(d.presentation.Slides)
	if len(layoutsForAppendPages) > 0 {
		layoutMap := d.layoutMap()
		var layoutObjectIDs = make([]string, len(layoutsForAppendPages))
		for i, l := range layoutsForAppendPages {
			layout, ok := layoutMap[l]
			if !ok {
				return fmt.Errorf("layout not found: %q", l)
			}
			layoutObjectIDs[i] = layout.ObjectId
		}
		// prepare pages for appending new slides in advance
		if err := d.preparePages(ctx, currentSlidesLen, layoutObjectIDs); err != nil {
			return fmt.Errorf("failed to create pages: %w", err)
		}
	}

	var (
		nextAppendingIndex = currentSlidesLen
		deletingIndices    []int
		applyRequests      []*slides.Request
		appendingCount     = 0
		applyingCount      = 0
	)
	for _, action := range actions {
		if action.actionType != actionTypeAppend && action.actionType != actionTypeUpdate &&
			len(applyRequests) > 0 {

			if err := d.batchUpdate(ctx, applyRequests); err != nil {
				return fmt.Errorf("failed to apply pages in batches: %w", err)
			}

			// Fill table content for updated/appended slides
			if err := d.fillTableContentForActions(ctx, actions); err != nil {
				return err
			}
			if appendingCount > 0 {
				d.logger.Info("appended pages", slog.Int("count", appendingCount))
				appendingCount = 0
			}
			if applyingCount > 0 {
				d.logger.Info("applied pages", slog.Int("count", applyingCount))
				applyingCount = 0
			}
			applyRequests = nil
		}
		if action.actionType != actionTypeDelete && len(deletingIndices) > 0 {
			// The indexes of consecutive delete actions are sorted in descending order,
			// so no position adjustment is necessary.
			if err := d.DeletePages(ctx, deletingIndices); err != nil {
				return fmt.Errorf("failed to delete pages: %w", err)
			}
			deletingIndices = nil
		}
		switch action.actionType {
		case actionTypeAppend:
			d.logger.Info("preparing to append new page")
			if reqs, err := d.prepareToApplyPage(ctx, nextAppendingIndex, action.slide, nil); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			} else if len(reqs) > 0 {
				applyRequests = append(applyRequests, reqs...)
			}
			appendingCount++
			nextAppendingIndex++
		case actionTypeUpdate:
			d.logger.Info("preparing to apply page", slog.Int("index", action.index))
			if reqs, err := d.prepareToApplyPage(ctx, action.index, action.slide, currentImages[action.index]); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			} else if len(reqs) > 0 {
				applyRequests = append(applyRequests, reqs...)
			}
			applyingCount++
		case actionTypeMove:
			if err := d.MovePage(ctx, action.index, action.moveToIndex); err != nil {
				return fmt.Errorf("failed to move page: %w", err)
			}
		case actionTypeDelete:
			deletingIndices = append(deletingIndices, action.index)
		}
	}
	if len(applyRequests) > 0 {
		if err := d.batchUpdate(ctx, applyRequests); err != nil {
			return fmt.Errorf("failed to apply pages in batches: %w", err)
		}

		// Fill table content for updated/appended slides
		if err := d.fillTableContentForActions(ctx, actions); err != nil {
			return err
		}

		if appendingCount > 0 {
			d.logger.Info("appended pages", slog.Int("count", appendingCount))
		}
		if applyingCount > 0 {
			d.logger.Info("applied pages", slog.Int("count", applyingCount))
		}
	}
	if len(deletingIndices) > 0 {
		return d.DeletePages(ctx, deletingIndices)
	}
	return d.refresh(ctx)
}

func (d *Deck) batchUpdate(ctx context.Context, requests []*slides.Request) error {
	d.logger.Info("batch updating presentation request", slog.Int("count", len(requests)))
	// Although there is no explicit request limit specified in the Google Slides API specifications,
	// we will set an upper limit as a precaution.
	// After testing several times, it handles around 1,000 requests without any issues so that we will
	// set the upper limit at that point for now.
	// This limit corresponds to approximately 100 pages of presentation requests.
	const reqCountLimit = 1000
	reqLen := len(requests)
	var groups [][]*slides.Request
	for i := 0; i < reqLen; i += reqCountLimit {
		end := min(i+reqCountLimit, reqLen)
		groups = append(groups, requests[i:end])
	}
	for _, requests := range groups {
		req := &slides.BatchUpdatePresentationRequest{
			Requests: requests,
		}
		if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
			return fmt.Errorf("failed to batch update presentation: %w", err)
		}
	}
	return nil
}

func (d *Deck) prepareToApplyPage(ctx context.Context, index int, slide *Slide, preloaded *currentImageData) (
	requests []*slides.Request, err error) {

	defer func() {
		err = errors.WithStack(err)
	}()

	layoutMap := d.layoutMap()
	layout, ok := layoutMap[slide.Layout]
	if !ok {
		return nil, fmt.Errorf("layout not found: %q", slide.Layout)
	}

	if len(d.presentation.Slides) <= index {
		return nil, fmt.Errorf("index out of range: %d", index)
	}
	if slide.Freeze {
		d.logger.Info("skip applying page. because freeze:true", slog.Int("index", index))
		return nil, nil
	}
	currentSlide := d.presentation.Slides[index]
	if currentSlide.SlideProperties.LayoutObjectId != layout.ObjectId {
		if err := d.updateLayout(ctx, index, slide); err != nil {
			return nil, err
		}
		// Reset preloaded data since layout has changed and internal page is changed.
		preloaded = nil
	}

	var (
		titles                    []placeholder
		subtitles                 []placeholder
		bodies                    []placeholder
		imagePlaceholders         []placeholder
		currentImages             []*Image
		currentImageObjectIDMap   = map[*Image]string{} // key: *Image, value: objectID
		currentTextBoxes          []*textBox
		currentTextBoxObjectIDMap = map[*textBox]string{} // key: *textBox, value: objectID
		currentTables             []*slides.PageElement
	)

	// Use preloaded image data if available, otherwise fetch on demand
	if preloaded != nil {
		currentImages = preloaded.currentImages
		currentImageObjectIDMap = preloaded.currentImageObjectIDMap
	}

	currentSlide = d.presentation.Slides[index]
	for _, element := range currentSlide.PageElements {
		switch {
		case element.Shape != nil && element.Shape.Placeholder != nil:
			switch element.Shape.Placeholder.Type {
			case "CENTERED_TITLE", "TITLE":
				titles = append(titles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				requests = append(requests, d.clearPlaceholderRequests(element)...)
			case "SUBTITLE":
				subtitles = append(subtitles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				requests = append(requests, d.clearPlaceholderRequests(element)...)
			case "BODY":
				bodies = append(bodies, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				requests = append(requests, d.clearPlaceholderRequests(element)...)
			}
		case element.Image != nil && element.Image.Placeholder != nil:
			imagePlaceholders = append(imagePlaceholders, placeholder{
				objectID: element.ObjectId,
				x:        element.Transform.TranslateX,
				y:        element.Transform.TranslateY,
			})
		case element.Image != nil && preloaded == nil:
			// Only fetch images on demand if preloaded data is not available
			var (
				image *Image
				err   error
			)
			if element.Description == descriptionImageFromMarkdown {
				image, err = NewImageFromMarkdown(element.Image.ContentUrl)
				if err != nil {
					return nil, fmt.Errorf("failed to create image from code block %s: %w", element.Image.ContentUrl, err)
				}
			} else {
				image, err = NewImage(element.Image.ContentUrl)
				if err != nil {
					return nil, fmt.Errorf("failed to create image from %s: %w", element.Image.ContentUrl, err)
				}
			}
			currentImages = append(currentImages, image)
			currentImageObjectIDMap[image] = element.ObjectId
		case element.Shape != nil && element.Shape.ShapeType == "TEXT_BOX" && element.Shape.Text != nil:
			tb := &textBox{}
			if element.Description == descriptionTextboxFromMarkdown {
				tb.fromMarkdown = true
			}
			tb.paragraphs = convertToParagraphs(element.Shape.Text)
			currentTextBoxes = append(currentTextBoxes, tb)
			currentTextBoxObjectIDMap[tb] = element.ObjectId
		case element.Table != nil:
			currentTables = append(currentTables, element)
		}
	}
	var speakerNotesID string
	for _, element := range currentSlide.SlideProperties.NotesPage.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			if element.Shape.Placeholder.Type == "BODY" {
				speakerNotesID = element.ObjectId
				requests = append(requests, d.clearPlaceholderRequests(element)...)
			}
		}
	}
	if speakerNotesID == "" {
		return nil, fmt.Errorf("speaker notes not found")
	}

	// set titles
	sort.Slice(titles, func(i, j int) bool {
		if titles[i].y == titles[j].y {
			return titles[i].x < titles[j].x
		}
		return titles[i].y < titles[j].y
	})
	for i, b := range slide.TitleBodies {
		if len(titles) <= i {
			break
		}
		reqs, styleReqs, err := d.applyParagraphsRequests(titles[i].objectID, b.Paragraphs)
		if err != nil {
			return nil, fmt.Errorf("failed to apply paragraphs for title: %w", err)
		}
		requests = append(requests, reqs...)
		requests = append(requests, styleReqs...)
	}

	// set subtitles
	sort.Slice(subtitles, func(i, j int) bool {
		if subtitles[i].y == subtitles[j].y {
			return subtitles[i].x < subtitles[j].x
		}
		return subtitles[i].y < subtitles[j].y
	})
	for i, b := range slide.SubtitleBodies {
		if len(subtitles) <= i {
			break
		}
		reqs, styleReqs, err := d.applyParagraphsRequests(subtitles[i].objectID, b.Paragraphs)
		if err != nil {
			return nil, fmt.Errorf("failed to apply paragraphs for subtitle: %w", err)
		}
		requests = append(requests, reqs...)
		requests = append(requests, styleReqs...)
	}

	// set speaker notes
	requests = append(requests, &slides.Request{
		InsertText: &slides.InsertTextRequest{
			ObjectId: speakerNotesID,
			Text:     slide.SpeakerNote,
		},
	})

	// set bodies
	sort.Slice(bodies, func(i, j int) bool {
		if bodies[i].y == bodies[j].y {
			return bodies[i].x < bodies[j].x
		}
		return bodies[i].y < bodies[j].y
	})
	for i, body := range slide.Bodies {
		if len(bodies) <= i {
			continue
		}
		reqs, styleReqs, err := d.applyParagraphsRequests(bodies[i].objectID, body.Paragraphs)
		if err != nil {
			return nil, fmt.Errorf("failed to apply paragraphs: %w", err)
		}
		requests = append(requests, reqs...)
		requests = append(requests, styleReqs...)
	}

	// set images
	sort.Slice(imagePlaceholders, func(i, j int) bool {
		if imagePlaceholders[i].y == imagePlaceholders[j].y {
			return imagePlaceholders[i].x < imagePlaceholders[j].x
		}
		return imagePlaceholders[i].y < imagePlaceholders[j].y
	})
	for i, image := range slide.Images {
		found := slices.ContainsFunc(currentImages, func(currentImage *Image) bool {
			return currentImage.Equivalent(image)
		})
		if found {
			continue
		}

		// Wait for image upload to complete
		info, err := image.UploadInfo(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to upload image: %w", err)
		}
		if info == nil {
			return nil, fmt.Errorf("image not uploaded or webContentLink is empty")
		}
		var imageObjectID string
		if len(imagePlaceholders) > i {
			imageReplaceMethod := "CENTER_CROP"
			if info.codeBlock {
				// In the case of code blocks, it is important that the entire image can be seen
				// without being cropped, so switch the replace method.
				imageReplaceMethod = "CENTER_INSIDE"
			}
			imageObjectID = imagePlaceholders[i].objectID
			requests = append(requests, &slides.Request{
				ReplaceImage: &slides.ReplaceImageRequest{
					ImageObjectId:      imagePlaceholders[i].objectID,
					ImageReplaceMethod: imageReplaceMethod,
					Url:                info.url,
				},
			})
		} else {
			imageObjectID = fmt.Sprintf("image-%s", uuid.New().String())
			imageReq := &slides.CreateImageRequest{
				ObjectId: imageObjectID,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: currentSlide.ObjectId,
					Transform: &slides.AffineTransform{
						ScaleX:     1.0,
						ScaleY:     1.0,
						TranslateX: float64(i+1) * 100000,
						TranslateY: float64(i+1) * 100000,
						Unit:       "EMU",
					},
				},
				Url: info.url,
			}
			requests = append(requests, &slides.Request{
				CreateImage: imageReq,
			})
		}
		if image.fromMarkdown {
			requests = append(requests, &slides.Request{
				UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
					ObjectId:    imageObjectID,
					Description: descriptionImageFromMarkdown,
				},
			})
		}
	}

	// set tables - compare with existing and only create/update as needed
	tableRequests, err := d.handleTableUpdates(currentSlide.ObjectId, slide.Tables, currentTables)
	if err != nil {
		return nil, fmt.Errorf("failed to handle table updates: %w", err)
	}
	requests = append(requests, tableRequests...)

	// set text boxes
	for i, bq := range slide.BlockQuotes {
		found := slices.ContainsFunc(currentTextBoxes, func(currentTextBox *textBox) bool {
			return slices.EqualFunc(currentTextBox.paragraphs, bq.Paragraphs, paragraphEqual)
		})
		if found {
			continue
		}
		// create new text box
		textBoxObjectID := fmt.Sprintf("textbox-%s", uuid.New().String())
		requests = append(requests, &slides.Request{
			CreateShape: &slides.CreateShapeRequest{
				ObjectId: textBoxObjectID,
				ElementProperties: &slides.PageElementProperties{
					PageObjectId: currentSlide.ObjectId,
					Size: &slides.Size{
						Height: &slides.Dimension{
							Magnitude: float64(500000 * len(bq.Paragraphs)),
							Unit:      "EMU",
						},
						Width: &slides.Dimension{
							Magnitude: 5000000,
							Unit:      "EMU",
						},
					},
					Transform: &slides.AffineTransform{
						ScaleX:     1.0,
						ScaleY:     1.0,
						TranslateX: float64(i+1) * 100000,
						TranslateY: float64(i+1) * 100000,
						Unit:       "EMU",
					},
				},
				ShapeType: "TEXT_BOX",
			},
		})

		sp, ok := d.shapes[styleBlockQuote]
		if ok {
			requests = append(requests, &slides.Request{
				UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
					ObjectId:        textBoxObjectID,
					ShapeProperties: sp,
					Fields:          "shapeBackgroundFill,outline,shadow",
				},
			})
		}
		reqs, styleReqs, err := d.applyParagraphsRequests(textBoxObjectID, bq.Paragraphs)
		if err != nil {
			return nil, fmt.Errorf("failed to apply paragraphs: %w", err)
		}
		requests = append(requests, reqs...)

		s, ok := d.styles[styleBlockQuote]
		if ok {
			r := buildCustomStyleRequest(s)
			r.ObjectId = textBoxObjectID
			requests = append(requests, &slides.Request{
				UpdateTextStyle: r,
			})
		}

		requests = append(requests, styleReqs...)

		requests = append(requests, &slides.Request{
			UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
				ObjectId:    textBoxObjectID,
				Description: descriptionTextboxFromMarkdown,
			},
		})
	}

	// set skip flag to slide
	requests = append(requests, &slides.Request{
		UpdateSlideProperties: &slides.UpdateSlidePropertiesRequest{
			ObjectId: currentSlide.ObjectId,
			SlideProperties: &slides.SlideProperties{
				IsSkipped: slide.Skip,
			},
			Fields: "isSkipped",
		},
	})

	// prune unmatched images via markdown
	for _, currentImage := range currentImages {
		if !currentImage.fromMarkdown {
			continue
		}
		found := slices.ContainsFunc(slide.Images, func(image *Image) bool {
			return currentImage.Equivalent(image)
		})
		if found {
			continue
		}
		imageObjectID, ok := currentImageObjectIDMap[currentImage]
		if !ok {
			return nil, fmt.Errorf("image object ID not found for image: %s", currentImage.url)
		}
		requests = append(requests, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: imageObjectID,
			},
		})
	}

	// prune unmatched text boxes via markdown
	for _, currentTextBox := range currentTextBoxes {
		if !currentTextBox.fromMarkdown {
			continue
		}
		found := slices.ContainsFunc(slide.BlockQuotes, func(bq *BlockQuote) bool {
			return slices.EqualFunc(currentTextBox.paragraphs, bq.Paragraphs, paragraphEqual)
		})
		if found {
			continue
		}
		textBoxObjectID, ok := currentTextBoxObjectIDMap[currentTextBox]
		if !ok {
			return nil, fmt.Errorf("text box object ID not found for text box: %v", currentTextBox.paragraphs)
		}
		requests = append(requests, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: textBoxObjectID,
			},
		})
	}

	return requests, nil
}

func (d *Deck) applyParagraphsRequests(objectID string, paragraphs []*Paragraph) (reqs []*slides.Request, styleReqs []*slides.Request, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	bulletRanges := map[int]*bulletRange{}
	count := int64(0)
	text := ""
	bulletStartIndex := int64(0) // reset per body
	bulletEndIndex := int64(0)   // reset per body
	currentBullet := BulletNone
	for j, paragraph := range paragraphs {
		plen := 0
		if paragraph.Bullet != BulletNone {
			if paragraph.Nesting > 0 {
				text += strings.Repeat("\t", paragraph.Nesting)
				plen += paragraph.Nesting
			}
		}
		for _, fragment := range paragraph.Fragments {
			// In Google Slides, pressing Enter creates a paragraph break, and pressing Shift + Enter
			// creates an inline line break. The inline line break seems to be treated as a vertical
			// tab around API data, so convert it to a vertical tab.
			fValue := strings.ReplaceAll(fragment.Value, "\n", "\v")
			flen := countString(fragment.Value)

			if r := d.getInlineStyleRequest(fragment); r != nil {
				startIndex := count + int64(plen)
				styleReqs = append(styleReqs, &slides.Request{
					UpdateTextStyle: &slides.UpdateTextStyleRequest{
						ObjectId: objectID,
						Style:    r.Style,
						Fields:   r.Fields,
						TextRange: &slides.Range{
							Type:       "FIXED_RANGE",
							StartIndex: ptrInt64(startIndex),
							EndIndex:   ptrInt64(startIndex + int64(flen)),
						},
					},
				})
			}
			plen += flen
			text += fValue
		}

		if len(paragraphs) > j+1 {
			text += "\n"
			plen++
		}

		if paragraph.Bullet != BulletNone {
			if paragraph.Nesting == 0 && currentBullet != paragraph.Bullet {
				bulletStartIndex = count
				bulletEndIndex = count
				bulletRanges[int(bulletStartIndex)] = &bulletRange{
					bullet: paragraph.Bullet,
					start:  bulletStartIndex,
					end:    bulletEndIndex,
				}
			}
			bulletEndIndex += int64(plen)
			bulletRanges[int(bulletStartIndex)].end = bulletEndIndex
		}
		currentBullet = paragraph.Bullet
		count += int64(plen)
	}

	reqs = append(reqs, &slides.Request{
		InsertText: &slides.InsertTextRequest{
			ObjectId: objectID,
			Text:     text,
		},
	})
	var bulletRangeSlice []*bulletRange
	for _, r := range bulletRanges {
		bulletRangeSlice = append(bulletRangeSlice, r)
	}
	// reverse sort
	// Because the Range changes each time it is converted to a list, convert from the end to a list.
	sort.Slice(bulletRangeSlice, func(i, j int) bool {
		return bulletRangeSlice[i].start > bulletRangeSlice[j].start
	})
	for _, r := range bulletRangeSlice {
		startIndex := r.start
		endIndex := r.end - 1
		if startIndex <= endIndex {
			endIndex++
		}
		styleReqs = append(styleReqs, &slides.Request{
			CreateParagraphBullets: &slides.CreateParagraphBulletsRequest{
				ObjectId:     objectID,
				BulletPreset: convertBullet(r.bullet),
				TextRange: &slides.Range{
					Type:       "FIXED_RANGE",
					StartIndex: ptrInt64(startIndex),
					EndIndex:   ptrInt64(endIndex),
				},
			},
		})
	}

	return reqs, styleReqs, nil
}

func (d *Deck) clearPlaceholderRequests(elm *slides.PageElement) []*slides.Request {
	if elm.Shape.Text == nil {
		return nil
	}
	return []*slides.Request{{
		UpdateTextStyle: &slides.UpdateTextStyleRequest{
			ObjectId: elm.ObjectId,
			Style: &slides.TextStyle{
				Bold:   false,
				Italic: false,
			},
			TextRange: &slides.Range{
				Type: "ALL",
			},
			Fields: "*",
		},
	}, {
		DeleteParagraphBullets: &slides.DeleteParagraphBulletsRequest{
			ObjectId: elm.ObjectId,
			TextRange: &slides.Range{
				Type: "ALL",
			},
		},
	}, {
		DeleteText: &slides.DeleteTextRequest{
			ObjectId: elm.ObjectId,
			TextRange: &slides.Range{
				Type: "ALL",
			},
		},
	}}
}

// countString counts the number of characters in a string, considering UTF-16 surrogate pairs.
// This is because Google Slides' character count is derived from JavaScript.
func countString(s string) int {
	length := 0
	for _, r := range s {
		if r <= 0xFFFF && (r < 0xD800 || r > 0xDFFF) {
			length++
		} else {
			length += 2
		}
	}
	return length
}

func ptrInt64(i int64) *int64 {
	return &i
}

func convertBullet(b Bullet) string {
	switch b {
	case BulletDash:
		return "BULLET_DISC_CIRCLE_SQUARE"
	case BulletNumber:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	case BulletAlpha:
		return "NUMBERED_DIGIT_ALPHA_ROMAN"
	default:
		return "UNRECOGNIZED"
	}
}

// getBulletPresetFromSlidesBullet converts a slides.Bullet to a BulletPreset string.
func getBulletPresetFromSlidesBullet(bullet *slides.Bullet) Bullet {
	if bullet == nil || bullet.Glyph == "" {
		return BulletNone
	}

	glyph := bullet.Glyph
	// Check for numbered bullets (1, 2, 3, etc.)
	for _, digit := range "0123456789" {
		if strings.Contains(glyph, string(digit)) {
			return BulletNumber
		}
	}

	// Check for alphabetic bullets (a., A., etc.)
	if strings.ContainsAny(glyph, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return BulletAlpha
	}

	// Default to disc/circle/square bullets
	return BulletDash
}

func (d *Deck) updateLayout(ctx context.Context, index int, slide *Slide) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()
	currentSlide := d.presentation.Slides[index]
	// create new page
	if err := d.createPage(ctx, index+1, slide); err != nil {
		return err
	}

	newSlide := d.presentation.Slides[index+1]
	var (
		reqs       []*slides.Request
		insertReqs []*slides.Request
		styleReqs  []*slides.Request
		bulletReqs []*slides.Request
	)

	for _, element := range currentSlide.PageElements {
		// copy images from the current slide to the new slide
		if element.Image != nil && element.Image.ContentUrl != "" {
			var imageObjectID string
			if element.Description == descriptionImageFromMarkdown {
				imageObjectID = fmt.Sprintf("image-%s", uuid.New().String())
			}
			reqs = append(reqs, &slides.Request{
				CreateImage: &slides.CreateImageRequest{
					ObjectId: imageObjectID,
					ElementProperties: &slides.PageElementProperties{
						Size:         element.Size,
						Transform:    element.Transform,
						PageObjectId: newSlide.ObjectId,
					},
					Url: element.Image.ContentUrl,
				},
			})
			if imageObjectID != "" {
				reqs = append(reqs, &slides.Request{
					UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
						ObjectId:    imageObjectID,
						Description: descriptionImageFromMarkdown,
					},
				})
			}
		}
		// copy shapes from the current slide to the new slide
		if element.Shape != nil && element.Shape.Placeholder == nil && element.Description != descriptionTextboxFromMarkdown {
			type paragraphInfo struct {
				startIndex   int64
				endIndex     int64
				bullet       *slides.Bullet
				nestingLevel int64
			}

			var paragraphInfos []paragraphInfo
			currentIndex := int64(0)
			text := ""
			shapeObjectID := fmt.Sprintf("shape-%s", uuid.New().String())

			for _, textElement := range element.Shape.Text.TextElements {
				if textElement.ParagraphMarker != nil {
					pInfo := paragraphInfo{
						startIndex: currentIndex,
					}
					if textElement.ParagraphMarker.Bullet != nil {
						pInfo.bullet = textElement.ParagraphMarker.Bullet
						pInfo.nestingLevel = textElement.ParagraphMarker.Bullet.NestingLevel
					}
					paragraphInfos = append(paragraphInfos, pInfo)
				}

				if textElement.TextRun != nil {
					runText := textElement.TextRun.Content

					// Handle nesting by adding tabs
					if len(paragraphInfos) > 0 && currentIndex == paragraphInfos[len(paragraphInfos)-1].startIndex {
						// This is the start of a bulleted paragraph
						if paragraphInfos[len(paragraphInfos)-1].nestingLevel > 0 {
							// Add tabs for nesting
							tabs := strings.Repeat("\t", int(paragraphInfos[len(paragraphInfos)-1].nestingLevel))
							text += tabs
							currentIndex += int64(countString(tabs))
						}
					}

					text += runText

					// Adjust style indices based on actual position in new text
					if textElement.TextRun.Style != nil {
						startIdx := currentIndex
						endIdx := currentIndex + int64(countString(runText))
						styleReqs = append(styleReqs, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId: shapeObjectID,
								Style:    textElement.TextRun.Style,
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: ptrInt64(startIdx),
									EndIndex:   ptrInt64(endIdx),
								},
								Fields: "*",
							},
						})
					}
					currentIndex += int64(countString(runText))
				}
			}

			// Update end indices for paragraphs
			for i := range paragraphInfos {
				if i < len(paragraphInfos)-1 {
					paragraphInfos[i].endIndex = paragraphInfos[i+1].startIndex - 1
				} else {
					paragraphInfos[i].endIndex = currentIndex - 1
				}
			}
			reqs = append(reqs, &slides.Request{
				CreateShape: &slides.CreateShapeRequest{
					ObjectId: shapeObjectID,
					ElementProperties: &slides.PageElementProperties{
						Size:         element.Size,
						Transform:    element.Transform,
						PageObjectId: newSlide.ObjectId,
					},
					ShapeType: element.Shape.ShapeType,
				},
			})
			styleReqs = append(styleReqs, &slides.Request{
				UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
					ObjectId:        shapeObjectID,
					ShapeProperties: element.Shape.ShapeProperties,
					Fields:          "contentAlignment,link,outline,shadow,shapeBackgroundFill",
				},
			})

			insertReqs = append(insertReqs, &slides.Request{
				InsertText: &slides.InsertTextRequest{
					ObjectId: shapeObjectID,
					Text:     strings.TrimSuffix(text, "\n"),
				},
			})

			var br *bulletRange
			for _, pInfo := range paragraphInfos {
				if pInfo.bullet == nil {
					if br != nil {
						bulletReqs = append(bulletReqs, &slides.Request{
							CreateParagraphBullets: &slides.CreateParagraphBulletsRequest{
								ObjectId:     shapeObjectID,
								BulletPreset: convertBullet(br.bullet),
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: ptrInt64(br.start),
									EndIndex:   ptrInt64(br.end),
								},
							},
						})
						br = nil
					}
					continue
				}
				if br == nil {
					br = &bulletRange{
						bullet: getBulletPresetFromSlidesBullet(pInfo.bullet),
						start:  pInfo.startIndex,
						end:    pInfo.endIndex,
					}
				} else {
					br.end = pInfo.endIndex
				}
			}
			if br != nil {
				bulletReqs = append(bulletReqs, &slides.Request{
					CreateParagraphBullets: &slides.CreateParagraphBulletsRequest{
						ObjectId:     shapeObjectID,
						BulletPreset: convertBullet(br.bullet),
						TextRange: &slides.Range{
							Type:       "FIXED_RANGE",
							StartIndex: ptrInt64(br.start),
							EndIndex:   ptrInt64(br.end),
						},
					},
				})
			}

			// reverse sort
			// Because the Range changes each time it is converted to a list, convert from the end to a list.
			sort.Slice(bulletReqs, func(i, j int) bool {
				return *bulletReqs[i].CreateParagraphBullets.TextRange.StartIndex > *bulletReqs[j].CreateParagraphBullets.TextRange.StartIndex
			})

			if len(styleReqs) > 0 || len(bulletReqs) > 0 {
				// Apply styles first, then bullets (important for correct rendering)
				insertReqs = append(insertReqs, styleReqs...)
				insertReqs = append(insertReqs, bulletReqs...)
				styleReqs = nil  // reset after adding to requests
				bulletReqs = nil // reset after adding to requests
			}
		}
	}
	reqs = append(reqs, insertReqs...)
	if len(reqs) > 0 {
		if err := d.batchUpdate(ctx, reqs); err != nil {
			return fmt.Errorf("failed to copy images or insert text: %w", err)
		}
	}
	if err := d.DeletePages(ctx, []int{index}); err != nil {
		return err
	}
	return nil
}
