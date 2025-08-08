// convert.go contains the processing to convert Google Slides data structures into internal deck data structures.
package deck

import (
	"strings"

	"google.golang.org/api/slides/v1"
)

func convertToSlide(p *slides.Page, layoutObjectIdMap map[string]*slides.Page) *Slide {
	slide := &Slide{
		Layout: "",
		Freeze: false,
	}
	if p.SlideProperties != nil {
		page, ok := layoutObjectIdMap[p.SlideProperties.LayoutObjectId]
		if ok {
			slide.Layout = page.LayoutProperties.DisplayName
		}
	}

	var titles []string
	var subtitles []string
	var bodies []*Body
	var images []*Image
	var blockQuotes []*BlockQuote
	var tables []*Table

	// Extract titles, subtitles, and bodies from page elements
	for _, element := range p.PageElements {
		switch {
		case element.Shape != nil && element.Shape.Text != nil && element.Shape.Placeholder != nil:
			switch element.Shape.Placeholder.Type {
			case "CENTERED_TITLE", "TITLE":
				text := extractText(element.Shape.Text)
				if text != "" {
					titles = append(titles, text)
				}
			case "SUBTITLE":
				text := extractText(element.Shape.Text)
				if text != "" {
					subtitles = append(subtitles, text)
				}
			case "BODY":
				paragraphs := convertToParagraphs(element.Shape.Text)
				if len(paragraphs) > 0 {
					bodies = append(bodies, &Body{
						Paragraphs: paragraphs,
					})
				}
			}
		case element.Image != nil:
			var (
				image *Image
				err   error
			)
			if element.Description == descriptionImageFromMarkdown {
				image, err = NewImageFromMarkdown(element.Image.ContentUrl)
				if err != nil {
					continue // Skip if image cannot be created
				}
			} else {
				image, err = NewImage(element.Image.ContentUrl)
				if err != nil {
					continue // Skip if image cannot be created
				}
			}
			images = append(images, image)
		case element.Shape != nil && element.Shape.ShapeType == "TEXT_BOX" && element.Shape.Text != nil:
			if element.Description != descriptionTextboxFromMarkdown {
				continue
			}
			bq := &BlockQuote{
				Paragraphs: convertToParagraphs(element.Shape.Text),
			}
			blockQuotes = append(blockQuotes, bq)
		case element.Table != nil:
			// Convert Google Slides table to deck Table
			table := convertSlidesToTable(element.Table)
			if table != nil {
				tables = append(tables, table)
			}
		}
	}

	slide.Titles = titles
	slide.Subtitles = subtitles
	slide.Bodies = bodies
	slide.Images = images
	slide.BlockQuotes = blockQuotes
	slide.Tables = tables

	// Extract speaker notes
	if p.SlideProperties != nil && p.SlideProperties.NotesPage != nil {
		for _, element := range p.SlideProperties.NotesPage.PageElements {
			if element.Shape != nil && element.Shape.Text != nil && element.Shape.Placeholder != nil {
				if element.Shape.Placeholder.Type == "BODY" {
					slide.SpeakerNote = extractText(element.Shape.Text)
					break
				}
			}
		}
	}

	return slide
}

// extractText extracts plain text from Shape.Text.
func extractText(text *slides.TextContent) string {
	if text == nil || len(text.TextElements) == 0 {
		return ""
	}

	var result strings.Builder
	for _, element := range text.TextElements {
		if element.TextRun != nil {
			result.WriteString(element.TextRun.Content)
		}
	}
	str := strings.ReplaceAll(result.String(), "\v", "\n")
	return strings.TrimSpace(str)
}

// convertToParagraphs converts TextContent to a slice of Paragraphs.
func convertToParagraphs(text *slides.TextContent) []*Paragraph {
	if text == nil || len(text.TextElements) == 0 {
		return nil
	}

	var paragraphs []*Paragraph
	var currentParagraph *Paragraph
	var currentBullet Bullet

	for _, element := range text.TextElements {

		switch {
		case element.ParagraphMarker != nil:
			// Start of a new paragraph
			if currentParagraph != nil && len(currentParagraph.Fragments) > 0 {
				paragraphs = append(paragraphs, currentParagraph)
			}
			currentParagraph = &Paragraph{
				Fragments: []*Fragment{},
				Nesting:   0,
			}

			// Process bullet points
			if element.ParagraphMarker.Bullet != nil {
				// Determine the type of bullet points based on glyph content
				if element.ParagraphMarker.Bullet.Glyph != "" {
					glyph := element.ParagraphMarker.Bullet.Glyph
					// Check for numbered bullets (1, 2, 3, etc.)
					if strings.ContainsAny(glyph, "0123456789") {
						currentBullet = BulletNumber
					} else {
						currentBullet = BulletDash
					}
				} else {
					// If no glyph, assume it's a dash bullet
					currentBullet = BulletDash
				}
				currentParagraph.Bullet = currentBullet

				// Set nesting level
				currentParagraph.Nesting = int(element.ParagraphMarker.Bullet.NestingLevel)
			} else {
				currentBullet = BulletNone
				currentParagraph.Bullet = currentBullet
			}
		case element.TextRun != nil:
			if currentParagraph == nil {
				continue
			}
			if frag := convertTextRunToFragment(element.TextRun); frag != nil {
				currentParagraph.Fragments = append(currentParagraph.Fragments, frag)
			}
		case element.AutoText != nil:
			// Only one of ParagraphMarker, TextRun, or AutoText in the element's properties will be non-nil.
			// Currently, nothing happens with AutoText, but we will prepare a branch just in case.
		}
	}

	// Add the last paragraph
	if currentParagraph != nil && len(currentParagraph.Fragments) > 0 {
		paragraphs = append(paragraphs, currentParagraph)
	}

	return paragraphs
}

func convertTextRunToFragment(textRun *slides.TextRun) *Fragment {
	// Get styles from TextRun
	var bold, italic, code bool
	var link string
	if textRun.Style != nil {
		bold = textRun.Style.Bold
		italic = textRun.Style.Italic
		if textRun.Style.Link != nil && textRun.Style.Link.Url != "" {
			link = textRun.Style.Link.Url
		}

		// Detect code style (based on font family and background color)
		if textRun.Style.FontFamily == defaultCodeFontFamily ||
			(textRun.Style.BackgroundColor != nil &&
				textRun.Style.BackgroundColor.OpaqueColor != nil &&
				textRun.Style.BackgroundColor.OpaqueColor.RgbColor != nil) {
			code = true
		}
	}

	content := textRun.Content
	// When checking the API response, a newline is always added to the end of the value of the
	// TextRun element before the modified paragraph, but since it is not necessary for the
	// information structure, we will delete it.
	content = strings.TrimSuffix(content, "\n")

	// When checking the API response, inline line breaks seem to be converted as vertical tabs,
	// so we will normalize them to line breaks.
	content = strings.ReplaceAll(content, "\v", "\n")
	if content == "" {
		return nil
	}
	return &Fragment{
		Value:  content,
		Bold:   bold,
		Italic: italic,
		Code:   code,
		Link:   link,
	}
}

// convertSlidesToTable converts a Google Slides table to deck Table structure.
func convertSlidesToTable(slidesTable *slides.Table) *Table {
	if slidesTable == nil || len(slidesTable.TableRows) == 0 {
		return nil
	}

	table := &Table{
		Rows: make([]*TableRow, len(slidesTable.TableRows)),
	}

	for i, slidesRow := range slidesTable.TableRows {
		if slidesRow == nil {
			continue
		}

		row := &TableRow{
			Cells: make([]*TableCell, len(slidesRow.TableCells)),
		}

		for j, slidesCell := range slidesRow.TableCells {
			if slidesCell == nil {
				continue
			}

			row.Cells[j] = &TableCell{
				Fragments: extractFragmentsFromTableCell(slidesCell),
				Alignment: extractAlignmentFromTableCell(slidesCell),
				IsHeader:  i == 0,
			}
		}

		table.Rows[i] = row
	}

	return table
}

// extractFragmentsFromTableCell extracts text fragments from a table cell.
func extractFragmentsFromTableCell(cell *slides.TableCell) []*Fragment {
	if cell.Text == nil || len(cell.Text.TextElements) == 0 {
		return nil
	}

	var fragments []*Fragment

	for _, element := range cell.Text.TextElements {
		if element.TextRun != nil {
			if frag := convertTextRunToFragment(element.TextRun); frag != nil {
				fragments = append(fragments, frag)
			}
		}
	}

	return fragments
}

// extractAlignmentFromTableCell extracts text alignment from a table cell.
func extractAlignmentFromTableCell(cell *slides.TableCell) string {
	if cell.Text == nil || len(cell.Text.TextElements) == 0 {
		return ""
	}
	// Look for paragraph style in the first text element
	for _, element := range cell.Text.TextElements {
		if element.ParagraphMarker != nil && element.ParagraphMarker.Style != nil {
			return element.ParagraphMarker.Style.Alignment
		}
	}
	return ""
}
