package deck

import (
	"strings"

	"google.golang.org/api/slides/v1"
)

// TableStyle holds the style information for table cells.
type TableStyle struct {
	HeaderFirstCol  *TableCellStyle // Style for header row, first column
	HeaderOtherCols *TableCellStyle // Style for header row, other columns
	DataFirstCol    *TableCellStyle // Style for data rows, first column
	DataOtherCols   *TableCellStyle // Style for data rows, other columns
	// Border styles
	BorderStyle *TableBorderStyle
}

// TableCellStyle holds style information for a single table cell.
type TableCellStyle struct {
	BackgroundFill   *slides.TableCellBackgroundFill
	TextStyle        *slides.TextStyle
	ContentAlignment string              // Vertical alignment: TOP, MIDDLE, BOTTOM
	ParagraphStyle   *slides.ParagraphStyle // Horizontal alignment, etc.
}

// TableBorderStyle holds border style information extracted from 2x2 table.
// Each field corresponds to a specific border position in the style template.
type TableBorderStyle struct {
	// Outer borders (from cell [0,0])
	OuterHorizontal *slides.TableBorderProperties // [0,0] Top -> outer top/bottom
	OuterVertical   *slides.TableBorderProperties // [0,0] Left -> outer left/right

	// Header row borders
	HeaderFirstColRight  *slides.TableBorderProperties // [0,0] Right -> header row, col 0 right
	HeaderFirstColBottom *slides.TableBorderProperties // [0,0] Bottom -> header row, col 0 bottom
	HeaderOtherColRight  *slides.TableBorderProperties // [0,1] Right -> header row, col 1+ right (except outer)
	HeaderOtherColBottom *slides.TableBorderProperties // [0,1] Bottom -> header row, col 1+ bottom

	// Data row borders
	DataFirstColRight  *slides.TableBorderProperties // [1,0] Right -> data rows, col 0 right
	DataFirstColBottom *slides.TableBorderProperties // [1,0] Bottom -> data rows, col 0 bottom (except outer)
	DataOtherColRight  *slides.TableBorderProperties // [1,1] Right -> data rows, col 1+ right (except outer)
	DataOtherColBottom *slides.TableBorderProperties // [1,1] Bottom -> data rows, col 1+ bottom (except outer)
}

// cellStyle returns the appropriate cell style based on row and column index.
func (ts *TableStyle) cellStyle(rowIdx, colIdx int) *TableCellStyle {
	if ts == nil {
		return nil
	}

	if rowIdx == 0 {
		// Header row
		if colIdx == 0 {
			return ts.HeaderFirstCol
		}
		return ts.HeaderOtherCols
	}
	// Data rows
	if colIdx == 0 {
		return ts.DataFirstCol
	}
	return ts.DataOtherCols
}

// defaultTableStyle returns the default table style (current hardcoded behavior).
func defaultTableStyle() *TableStyle {
	// Existing hardcoded header background color RGB(0.95, 0.95, 0.95)
	headerBg := &slides.TableCellBackgroundFill{
		SolidFill: &slides.SolidFill{
			Color: &slides.OpaqueColor{
				RgbColor: &slides.RgbColor{
					Red:   0.95,
					Green: 0.95,
					Blue:  0.95,
				},
			},
		},
	}

	// Header text style: bold
	headerTextStyle := &slides.TextStyle{
		Bold: true,
	}

	return &TableStyle{
		HeaderFirstCol:  &TableCellStyle{BackgroundFill: headerBg, TextStyle: headerTextStyle},
		HeaderOtherCols: &TableCellStyle{BackgroundFill: headerBg, TextStyle: headerTextStyle},
		DataFirstCol:    &TableCellStyle{},
		DataOtherCols:   &TableCellStyle{},
	}
}

// extractTableStyleFromLayout extracts table style from a 2x2 table in the style layout.
// Returns nil if the table is not 2x2.
func extractTableStyleFromLayout(table *slides.Table) *TableStyle {
	// Verify table is 2x2
	if len(table.TableRows) != 2 {
		return nil
	}
	for _, row := range table.TableRows {
		if len(row.TableCells) != 2 {
			return nil
		}
	}

	// Extract styles from each cell
	ts := &TableStyle{
		HeaderFirstCol:  extractCellStyle(table.TableRows[0].TableCells[0]),
		HeaderOtherCols: extractCellStyle(table.TableRows[0].TableCells[1]),
		DataFirstCol:    extractCellStyle(table.TableRows[1].TableCells[0]),
		DataOtherCols:   extractCellStyle(table.TableRows[1].TableCells[1]),
	}

	// Extract border styles
	ts.BorderStyle = extractBorderStyle(table)

	return ts
}

// extractBorderStyle extracts border styles from a 2x2 table.
// The 2x2 table has:
// - HorizontalBorderRows: 3 rows × 2 cols (top of row0, between row0/row1, bottom of row1).
// - VerticalBorderRows: 2 rows × 3 cols (left of col0, between col0/col1, right of col1).
func extractBorderStyle(table *slides.Table) *TableBorderStyle {
	bs := &TableBorderStyle{}

	// Extract horizontal borders
	// HorizontalBorderRows[0] = top border of row 0
	// HorizontalBorderRows[1] = bottom border of row 0 (= top border of row 1)
	// HorizontalBorderRows[2] = bottom border of row 1
	if len(table.HorizontalBorderRows) >= 3 {
		// [0,0] Top -> OuterHorizontal (for outer top/bottom)
		if row := table.HorizontalBorderRows[0]; row != nil && len(row.TableBorderCells) > 0 {
			if cell := row.TableBorderCells[0]; cell != nil {
				bs.OuterHorizontal = cell.TableBorderProperties
			}
		}
		// [0,0] Bottom -> HeaderFirstColBottom
		if row := table.HorizontalBorderRows[1]; row != nil && len(row.TableBorderCells) > 0 {
			if cell := row.TableBorderCells[0]; cell != nil {
				bs.HeaderFirstColBottom = cell.TableBorderProperties
			}
		}
		// [0,1] Bottom -> HeaderOtherColBottom
		if row := table.HorizontalBorderRows[1]; row != nil && len(row.TableBorderCells) > 1 {
			if cell := row.TableBorderCells[1]; cell != nil {
				bs.HeaderOtherColBottom = cell.TableBorderProperties
			}
		}
		// [1,0] Bottom -> DataFirstColBottom
		if row := table.HorizontalBorderRows[2]; row != nil && len(row.TableBorderCells) > 0 {
			if cell := row.TableBorderCells[0]; cell != nil {
				bs.DataFirstColBottom = cell.TableBorderProperties
			}
		}
		// [1,1] Bottom -> DataOtherColBottom
		if row := table.HorizontalBorderRows[2]; row != nil && len(row.TableBorderCells) > 1 {
			if cell := row.TableBorderCells[1]; cell != nil {
				bs.DataOtherColBottom = cell.TableBorderProperties
			}
		}
	}

	// Extract vertical borders
	// VerticalBorderRows[0] = borders for row 0 (left of col0, between col0/col1, right of col1)
	// VerticalBorderRows[1] = borders for row 1
	if len(table.VerticalBorderRows) >= 2 {
		// [0,0] Left -> OuterVertical (for outer left/right)
		if row := table.VerticalBorderRows[0]; row != nil && len(row.TableBorderCells) > 0 {
			if cell := row.TableBorderCells[0]; cell != nil {
				bs.OuterVertical = cell.TableBorderProperties
			}
		}
		// [0,0] Right -> HeaderFirstColRight
		if row := table.VerticalBorderRows[0]; row != nil && len(row.TableBorderCells) > 1 {
			if cell := row.TableBorderCells[1]; cell != nil {
				bs.HeaderFirstColRight = cell.TableBorderProperties
			}
		}
		// [0,1] Right -> HeaderOtherColRight
		if row := table.VerticalBorderRows[0]; row != nil && len(row.TableBorderCells) > 2 {
			if cell := row.TableBorderCells[2]; cell != nil {
				bs.HeaderOtherColRight = cell.TableBorderProperties
			}
		}
		// [1,0] Right -> DataFirstColRight
		if row := table.VerticalBorderRows[1]; row != nil && len(row.TableBorderCells) > 1 {
			if cell := row.TableBorderCells[1]; cell != nil {
				bs.DataFirstColRight = cell.TableBorderProperties
			}
		}
		// [1,1] Right -> DataOtherColRight
		if row := table.VerticalBorderRows[1]; row != nil && len(row.TableBorderCells) > 2 {
			if cell := row.TableBorderCells[2]; cell != nil {
				bs.DataOtherColRight = cell.TableBorderProperties
			}
		}
	}

	return bs
}

// extractCellStyle extracts style from a table cell.
func extractCellStyle(cell *slides.TableCell) *TableCellStyle {
	style := &TableCellStyle{}

	// Extract background color and content alignment
	if cell.TableCellProperties != nil {
		style.BackgroundFill = cell.TableCellProperties.TableCellBackgroundFill
		style.ContentAlignment = cell.TableCellProperties.ContentAlignment
	}

	// Extract text style and paragraph style (from first TextRun/ParagraphMarker)
	if cell.Text != nil {
		for _, te := range cell.Text.TextElements {
			if te.TextRun != nil && te.TextRun.Style != nil && style.TextStyle == nil {
				style.TextStyle = te.TextRun.Style
			}
			if te.ParagraphMarker != nil && te.ParagraphMarker.Style != nil && style.ParagraphStyle == nil {
				style.ParagraphStyle = te.ParagraphMarker.Style
			}
			// Break if both are found
			if style.TextStyle != nil && style.ParagraphStyle != nil {
				break
			}
		}
	}

	return style
}

// buildTableCellTextStyleRequest builds an UpdateTextStyleRequest from a TextStyle.
// Boolean fields (bold, italic, underline, strikethrough) are always included
// to allow explicitly setting them to false.
func buildTableCellTextStyleRequest(s *slides.TextStyle) *slides.UpdateTextStyleRequest {
	if s == nil {
		return nil
	}

	style := &slides.TextStyle{
		Bold:          s.Bold,
		Italic:        s.Italic,
		Underline:     s.Underline,
		Strikethrough: s.Strikethrough,
	}
	fields := []string{"bold", "italic", "underline", "strikethrough"}

	if s.FontFamily != "" {
		style.FontFamily = s.FontFamily
		fields = append(fields, "fontFamily")
	}
	if s.ForegroundColor != nil {
		style.ForegroundColor = s.ForegroundColor
		fields = append(fields, "foregroundColor")
	}
	if s.BackgroundColor != nil {
		style.BackgroundColor = s.BackgroundColor
		fields = append(fields, "backgroundColor")
	}
	if s.FontSize != nil {
		style.FontSize = s.FontSize
		fields = append(fields, "fontSize")
	}
	if s.BaselineOffset != "" {
		style.BaselineOffset = s.BaselineOffset
		fields = append(fields, "baselineOffset")
	}

	return &slides.UpdateTextStyleRequest{
		Style:  style,
		Fields: strings.Join(fields, ","),
	}
}
