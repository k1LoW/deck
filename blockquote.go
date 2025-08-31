package deck

import (
	"fmt"
	"slices"

	"github.com/google/uuid"
	"google.golang.org/api/slides/v1"
)

func (d *Deck) handleBlockquotes(
	objectId string, blockquotes []*BlockQuote, currentTextBoxes []*textBox, currentBlockquoteIDs []string) (
	requests []*slides.Request, reuseBlockquotes bool, err error) {

	reuseBlockquotes = len(currentBlockquoteIDs) == len(blockquotes)
	for i, bq := range blockquotes {
		if slices.ContainsFunc(currentTextBoxes, func(currentTextBox *textBox) bool {
			return slices.EqualFunc(currentTextBox.paragraphs, bq.Paragraphs, paragraphEqual)
		}) {
			continue
		}
		var textBoxObjectID string
		if reuseBlockquotes {
			textBoxObjectID = currentBlockquoteIDs[i]
			requests = append(requests, &slides.Request{
				DeleteText: &slides.DeleteTextRequest{
					ObjectId: textBoxObjectID,
					TextRange: &slides.Range{
						Type: "ALL",
					},
				},
			})
		} else {
			// create new text box
			textBoxObjectID = fmt.Sprintf("textbox-%s", uuid.New().String())
			requests = append(requests, &slides.Request{
				CreateShape: &slides.CreateShapeRequest{
					ObjectId: textBoxObjectID,
					ElementProperties: &slides.PageElementProperties{
						PageObjectId: objectId,
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
						// We want to specify `autofit.autofitType` (such as `SHAPE_AUTOFIT`), but we cannot specify it
						// because there is a problem with the Google Slide API.
						// See: https://issuetracker.google.com/issues/199176586
						Fields: "shapeBackgroundFill,outline,shadow",
					},
				})
			}
		}

		reqs, styleReqs, err := d.applyParagraphsRequests(textBoxObjectID, bq.Paragraphs)
		if err != nil {
			return nil, reuseBlockquotes, fmt.Errorf("failed to apply paragraphs: %w", err)
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
				Description: descriptionBlockquoteTextboxFromMarkdown,
			},
		})
	}
	return requests, reuseBlockquotes, nil
}
