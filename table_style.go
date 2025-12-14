package deck

import (
	"strings"

	"google.golang.org/api/slides/v1"
)

// TableStyle holds the style information for table cells.
type TableStyle struct {
	// Header row styles
	HeaderFirstCol  *TableCellStyle // Style for header row, first column
	HeaderOtherCols *TableCellStyle // Style for header row, other columns
	// Data row styles
	DataFirstCol  *TableCellStyle // Style for data rows, first column
	DataOtherCols *TableCellStyle // Style for data rows, other columns
}

// TableCellStyle holds style information for a single table cell.
type TableCellStyle struct {
	BackgroundFill *slides.TableCellBackgroundFill
	TextStyle      *slides.TextStyle
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
	return &TableStyle{
		HeaderFirstCol:  extractCellStyle(table.TableRows[0].TableCells[0]),
		HeaderOtherCols: extractCellStyle(table.TableRows[0].TableCells[1]),
		DataFirstCol:    extractCellStyle(table.TableRows[1].TableCells[0]),
		DataOtherCols:   extractCellStyle(table.TableRows[1].TableCells[1]),
	}
}

// extractCellStyle extracts style from a table cell.
func extractCellStyle(cell *slides.TableCell) *TableCellStyle {
	style := &TableCellStyle{}

	// Extract background color
	if cell.TableCellProperties != nil {
		style.BackgroundFill = cell.TableCellProperties.TableCellBackgroundFill
	}

	// Extract text style (from first TextRun)
	if cell.Text != nil {
		for _, te := range cell.Text.TextElements {
			if te.TextRun != nil && te.TextRun.Style != nil {
				style.TextStyle = te.TextRun.Style
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
