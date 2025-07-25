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
	styleCode                      = "code"
	styleBold                      = "bold"
	styleItalic                    = "italic"
	styleLink                      = "link"
	styleBlockQuote                = "blockquote"
	defaultCodeFontFamily          = "Noto Sans Mono"
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
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}
	layoutObjectIdMap := map[string]*slides.Page{}
	for _, l := range d.presentation.Layouts {
		layoutObjectIdMap[l.ObjectId] = l
	}

	before := make(Slides, 0, len(d.presentation.Slides))
	var after Slides
	for _, p := range d.presentation.Slides {
		slide := convertToSlide(p, layoutObjectIdMap)
		before = append(before, slide)
		after = append(after, slide)
	}

	for i, slide := range ss {
		if !slices.Contains(pages, i+1) {
			continue
		}
		if slide.Layout == "" {
			if i == 0 {
				slide.Layout = d.defaultTitleLayout
			} else {
				slide.Layout = d.defaultLayout
			}
		}
		if len(after) < i {
			after[i] = slide
		} else if len(after) == i {
			after = append(after, slide)
		} else {
			after[i] = slide
		}
	}
	if len(after) > len(ss) {
		var deleteIndexes []int
		for i := len(ss); i < len(after); i++ {
			if !slices.Contains(pages, i+1) {
				deleteIndexes = append(deleteIndexes, i)
			}
		}
		slices.Reverse(deleteIndexes)
		for _, i := range deleteIndexes {
			after = slices.Delete(after, i, i+1)
		}
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
		case actionTypeInsert:
			actionDetails = append(actionDetails, actionDetail{
				ActionType:  actionTypeInsert,
				Titles:      action.slide.Titles,
				Index:       &action.index,
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
	var deletingIndices []int
	for _, action := range actions {
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
			if err := d.AppendPage(ctx, action.slide); err != nil {
				return fmt.Errorf("failed to append slide: %w", err)
			}
		case actionTypeInsert:
			if err := d.InsertPage(ctx, action.index, action.slide); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			}
		case actionTypeUpdate:
			d.logger.Info("appling page", slog.Int("index", action.index))
			if err := d.applyPage(ctx, action.index, action.slide, currentImages[action.index]); err != nil {
				return fmt.Errorf("failed to apply page: %w", err)
			}
			d.logger.Info("applied page", slog.Int("index", action.index))
		case actionTypeMove:
			if err := d.MovePage(ctx, action.index, action.moveToIndex); err != nil {
				return fmt.Errorf("failed to move page: %w", err)
			}
		case actionTypeDelete:
			deletingIndices = append(deletingIndices, action.index)
		}
	}
	if len(deletingIndices) > 0 {
		return d.DeletePages(ctx, deletingIndices)
	}
	return nil
}

func (d *Deck) applyPage(ctx context.Context, index int, slide *Slide, preloaded *currentImageData) (err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	layoutMap := d.layoutMap()
	layout, ok := layoutMap[slide.Layout]
	if !ok {
		return fmt.Errorf("layout not found: %q", slide.Layout)
	}

	if len(d.presentation.Slides) <= index {
		return fmt.Errorf("index out of range: %d", index)
	}
	if slide.Freeze {
		d.logger.Info("skip applying page. because freeze:true", slog.Int("index", index))
		return nil
	}
	currentSlide := d.presentation.Slides[index]
	if currentSlide.SlideProperties.LayoutObjectId != layout.ObjectId {
		if err := d.updateLayout(ctx, index, slide); err != nil {
			return err
		}
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
		req                       = &slides.BatchUpdatePresentationRequest{}
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
				req.Requests = append(req.Requests, d.clearPlaceholderRequests(element)...)
			case "SUBTITLE":
				subtitles = append(subtitles, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				req.Requests = append(req.Requests, d.clearPlaceholderRequests(element)...)
			case "BODY":
				bodies = append(bodies, placeholder{
					objectID: element.ObjectId,
					x:        element.Transform.TranslateX,
					y:        element.Transform.TranslateY,
				})
				req.Requests = append(req.Requests, d.clearPlaceholderRequests(element)...)
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
					return fmt.Errorf("failed to create image from code block %s: %w", element.Image.ContentUrl, err)
				}
			} else {
				image, err = NewImage(element.Image.ContentUrl)
				if err != nil {
					return fmt.Errorf("failed to create image from %s: %w", element.Image.ContentUrl, err)
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
		}
	}
	var speakerNotesID string
	for _, element := range currentSlide.SlideProperties.NotesPage.PageElements {
		if element.Shape != nil && element.Shape.Placeholder != nil {
			if element.Shape.Placeholder.Type == "BODY" {
				speakerNotesID = element.ObjectId
				req.Requests = append(req.Requests, d.clearPlaceholderRequests(element)...)
			}
		}
	}
	if speakerNotesID == "" {
		return fmt.Errorf("speaker notes not found")
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
			return fmt.Errorf("failed to apply paragraphs for title: %w", err)
		}
		req.Requests = append(req.Requests, reqs...)
		req.Requests = append(req.Requests, styleReqs...)
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
			return fmt.Errorf("failed to apply paragraphs for subtitle: %w", err)
		}
		req.Requests = append(req.Requests, reqs...)
		req.Requests = append(req.Requests, styleReqs...)
	}

	// set speaker notes
	req.Requests = append(req.Requests, &slides.Request{
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
			return fmt.Errorf("failed to apply paragraphs: %w", err)
		}
		req.Requests = append(req.Requests, reqs...)
		req.Requests = append(req.Requests, styleReqs...)
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
			return currentImage.Compare(image)
		})
		if found {
			continue
		}

		// Wait for image upload to complete
		webContentLink, err := image.UploadInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to upload image: %w", err)
		}
		if webContentLink == "" {
			return fmt.Errorf("image not uploaded or webContentLink is empty")
		}
		var imageObjectID string
		if len(imagePlaceholders) > i {
			imageObjectID = imagePlaceholders[i].objectID
			req.Requests = append(req.Requests, &slides.Request{
				ReplaceImage: &slides.ReplaceImageRequest{
					ImageObjectId:      imagePlaceholders[i].objectID,
					ImageReplaceMethod: "CENTER_CROP",
					Url:                webContentLink,
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
				Url: webContentLink,
			}
			req.Requests = append(req.Requests, &slides.Request{
				CreateImage: imageReq,
			})
		}
		if image.fromMarkdown {
			req.Requests = append(req.Requests, &slides.Request{
				UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
					ObjectId:    imageObjectID,
					Description: descriptionImageFromMarkdown,
				},
			})
		}
	}

	// set text boxes
	for i, bq := range slide.BlockQuotes {
		found := slices.ContainsFunc(currentTextBoxes, func(currentTextBox *textBox) bool {
			return paragraphsEqual(currentTextBox.paragraphs, bq.Paragraphs)
		})
		if found {
			continue
		}
		// create new text box
		textBoxObjectID := fmt.Sprintf("textbox-%s", uuid.New().String())
		req.Requests = append(req.Requests, &slides.Request{
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
			req.Requests = append(req.Requests, &slides.Request{
				UpdateShapeProperties: &slides.UpdateShapePropertiesRequest{
					ObjectId:        textBoxObjectID,
					ShapeProperties: sp,
					Fields:          "shapeBackgroundFill,outline,shadow",
				},
			})
		}
		reqs, styleReqs, err := d.applyParagraphsRequests(textBoxObjectID, bq.Paragraphs)
		if err != nil {
			return fmt.Errorf("failed to apply paragraphs: %w", err)
		}
		req.Requests = append(req.Requests, reqs...)

		s, ok := d.styles[styleBlockQuote]
		if ok {
			req.Requests = append(req.Requests, &slides.Request{
				UpdateTextStyle: &slides.UpdateTextStyleRequest{
					ObjectId: textBoxObjectID,
					Style:    s,
					Fields:   "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
				},
			})
		}

		req.Requests = append(req.Requests, styleReqs...)

		req.Requests = append(req.Requests, &slides.Request{
			UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
				ObjectId:    textBoxObjectID,
				Description: descriptionTextboxFromMarkdown,
			},
		})
	}

	// set skip flag to slide
	req.Requests = append(req.Requests, &slides.Request{
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
			return currentImage.Compare(image)
		})
		if found {
			continue
		}
		imageObjectID, ok := currentImageObjectIDMap[currentImage]
		if !ok {
			return fmt.Errorf("image object ID not found for image: %s", currentImage.url)
		}
		req.Requests = append(req.Requests, &slides.Request{
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
			return paragraphsEqual(currentTextBox.paragraphs, bq.Paragraphs)
		})
		if found {
			continue
		}
		textBoxObjectID, ok := currentTextBoxObjectIDMap[currentTextBox]
		if !ok {
			return fmt.Errorf("text box object ID not found for text box: %v", currentTextBox.paragraphs)
		}
		req.Requests = append(req.Requests, &slides.Request{
			DeleteObject: &slides.DeleteObjectRequest{
				ObjectId: textBoxObjectID,
			},
		})
	}

	if len(req.Requests) > 0 {
		if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
			return err
		}
	}

	return nil
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

			var (
				fields string
				style  *slides.TextStyle
			)
			for _, r := range d.getInlineStyleRequests(fragment) {
				// Merge elements with the latter taking priority.
				fields = mergeFields(fields, r.Fields)
				style = mergeStyles(style, r.Style, r.Fields)
			}
			if style != nil {
				startIndex := count + int64(plen)
				styleReqs = append(styleReqs, &slides.Request{
					UpdateTextStyle: &slides.UpdateTextStyleRequest{
						ObjectId: objectID,
						Style:    style,
						Fields:   fields,
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
		startIndex := int64(r.start)
		endIndex := int64(r.end - 1)
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

func (d *Deck) getInlineStyleRequests(fragment *Fragment) (reqs []*slides.UpdateTextStyleRequest) {
	if fragment.Code {
		s, ok := d.styles[styleCode]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, &slides.UpdateTextStyleRequest{
				Style: &slides.TextStyle{
					ForegroundColor: &slides.OptionalColor{
						OpaqueColor: &slides.OpaqueColor{
							RgbColor: &slides.RgbColor{
								Red:   0.0,
								Green: 0.0,
								Blue:  0.0,
							},
						},
					},
					FontFamily: defaultCodeFontFamily,
					BackgroundColor: &slides.OptionalColor{
						OpaqueColor: &slides.OpaqueColor{
							RgbColor: &slides.RgbColor{
								Red:   0.95,
								Green: 0.95,
								Blue:  0.95,
							},
						},
					},
				},
				Fields: "foregroundColor,fontFamily,backgroundColor",
			})
		}
	}

	if fragment.Bold {
		s, ok := d.styles[styleBold]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, &slides.UpdateTextStyleRequest{
				Style: &slides.TextStyle{
					Bold: true,
				},
				Fields: "bold",
			})
		}
	}

	if fragment.Italic {
		s, ok := d.styles[styleItalic]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, &slides.UpdateTextStyleRequest{
				Style: &slides.TextStyle{
					Italic: true,
				},
				Fields: "italic",
			})
		}
	}

	if fragment.Link != "" {
		s, ok := d.styles[styleLink]
		if ok {
			req := buildCustomStyleRequest(s)
			req.Fields = "link,bold,italic,underline,foregroundColor,fontFamily,backgroundColor"
			req.Style.Link = &slides.Link{
				Url: fragment.Link,
			}
			reqs = append(reqs, req)
		} else {
			reqs = append(reqs, &slides.UpdateTextStyleRequest{
				Style: &slides.TextStyle{
					Link: &slides.Link{
						Url: fragment.Link,
					},
				},
				Fields: "link",
			})
		}
	}

	if fragment.StyleName != "" {
		s, ok := d.styles[fragment.StyleName]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		}
	}

	return reqs
}

func buildCustomStyleRequest(s *slides.TextStyle) *slides.UpdateTextStyleRequest {
	return &slides.UpdateTextStyleRequest{
		Style: &slides.TextStyle{
			Bold:            s.Bold,
			Italic:          s.Italic,
			Underline:       s.Underline,
			ForegroundColor: s.ForegroundColor,
			FontFamily:      s.FontFamily,
			BackgroundColor: s.BackgroundColor,
		},
		Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
	}
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

func mergeFields(a, b string) string {
	fields := strings.Split(a, ",")
	fields = append(fields, strings.Split(b, ",")...)
	sort.Strings(fields)
	fields = slices.Compact(fields)
	return strings.Join(fields, ",")
}

func mergeStyles(a, b *slides.TextStyle, fStr string) *slides.TextStyle {
	if a == nil {
		return b
	}
	fields := strings.Split(fStr, ",")
	if slices.Contains(fields, "link") {
		a.Link = b.Link
	}
	if slices.Contains(fields, "bold") {
		a.Bold = b.Bold
	}
	if slices.Contains(fields, "italic") {
		a.Italic = b.Italic
	}
	if slices.Contains(fields, "underline") {
		a.Underline = b.Underline
	}
	if slices.Contains(fields, "foregroundColor") {
		a.ForegroundColor = b.ForegroundColor
	}
	if slices.Contains(fields, "fontFamily") {
		a.FontFamily = b.FontFamily
	}
	if slices.Contains(fields, "backgroundColor") {
		a.BackgroundColor = b.BackgroundColor
	}
	return a
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
	req := &slides.BatchUpdatePresentationRequest{
		Requests: []*slides.Request{},
	}
	var (
		insertReqs []*slides.Request
		styleReqs  []*slides.Request
		bulletReqs []*slides.Request
	)

	for _, element := range currentSlide.PageElements {
		// copy images from the current slide to the new slide
		if element.Image != nil && element.Description != descriptionImageFromMarkdown && element.Image.ContentUrl != "" {
			req.Requests = append(req.Requests, &slides.Request{
				CreateImage: &slides.CreateImageRequest{
					ElementProperties: &slides.PageElementProperties{
						Size:         element.Size,
						Transform:    element.Transform,
						PageObjectId: newSlide.ObjectId,
					},
					Url: element.Image.ContentUrl,
				},
			})
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
			req.Requests = append(req.Requests, &slides.Request{
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
									StartIndex: ptrInt64(int64(br.start)),
									EndIndex:   ptrInt64(int64(br.end)),
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
							StartIndex: ptrInt64(int64(br.start)),
							EndIndex:   ptrInt64(int64(br.end)),
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
	req.Requests = append(req.Requests, insertReqs...)
	if len(req.Requests) > 0 {
		if _, err := d.srv.Presentations.BatchUpdate(d.id, req).Context(ctx).Do(); err != nil {
			return fmt.Errorf("failed to copy images or insert text: %w", err)
		}
	}

	if err := d.DeletePages(ctx, []int{index}); err != nil {
		return err
	}
	return nil
}
