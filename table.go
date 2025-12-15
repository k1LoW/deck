package deck

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/api/slides/v1"
)

const descriptionTableFromMarkdown = "Table generated from markdown"

func (d *Deck) handleTableUpdates(slideObjectID string, newTables []*Table, currentTables []*slides.PageElement) ([]*slides.Request, error) {
	var requests []*slides.Request

	// Filter only tables created by deck (with descriptionTableFromMarkdown)
	var deckTables []*slides.PageElement
	for _, element := range currentTables {
		if element.Table != nil && element.Description == descriptionTableFromMarkdown {
			deckTables = append(deckTables, element)
		}
	}

	// Convert existing deck-created tables to deck Tables for comparison
	var existingTables []*Table
	for _, element := range deckTables {
		table := convertSlidesToTable(element.Table)
		if table != nil {
			existingTables = append(existingTables, table)
		}
	}

	// Case 1: No tables needed, remove all existing deck-created tables
	if len(newTables) == 0 {
		for _, element := range deckTables {
			requests = append(requests, &slides.Request{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: element.ObjectId,
				},
			})
		}
		return requests, nil
	}

	// Case 2: Tables exist and we need to compare
	if tablesEqual(existingTables, newTables) {
		// Tables are identical, no action needed
		return nil, nil
	}

	// Case 3: Tables are different, need to update

	// Reuse existing deck-created tables where possible, adjusting their structure
	maxTables := max(len(deckTables), len(newTables))

	for i := range maxTables {
		if i < len(deckTables) && i < len(newTables) {
			// Reuse existing deck-created table: clear content and adjust structure
			tableReqs, err := d.reuseTableRequests(deckTables[i], newTables[i])
			if err != nil {
				return nil, fmt.Errorf("failed to reuse table %d: %w", i, err)
			}
			requests = append(requests, tableReqs...)
		} else if i < len(deckTables) {
			// Remove excess existing deck-created tables
			requests = append(requests, &slides.Request{
				DeleteObject: &slides.DeleteObjectRequest{
					ObjectId: deckTables[i].ObjectId,
				},
			})
		} else if i < len(newTables) {
			// Create new tables for additional ones needed
			createReqs, err := d.createTableStructureRequest(slideObjectID, newTables[i], i)
			if err != nil {
				return nil, fmt.Errorf("failed to create table structure request: %w", err)
			}
			requests = append(requests, createReqs...)
		}
	}

	return requests, nil
}

// reuseTableRequests creates requests to reuse an existing table by clearing its content and adjusting structure.
func (d *Deck) reuseTableRequests(existingElement *slides.PageElement, newTable *Table) ([]*slides.Request, error) {
	var requests []*slides.Request

	if existingElement.Table == nil {
		return nil, fmt.Errorf("existing element is not a table")
	}

	existingTable := existingElement.Table
	tableObjectID := existingElement.ObjectId

	// Calculate row and column differences
	existingRows := len(existingTable.TableRows)
	existingCols := 0
	// Find the maximum number of columns across all rows
	for _, row := range existingTable.TableRows {
		if row != nil && len(row.TableCells) > existingCols {
			existingCols = len(row.TableCells)
		}
	}

	newRows := len(newTable.Rows)
	newCols := 0
	// Find the maximum number of columns across all rows
	for _, row := range newTable.Rows {
		if len(row.Cells) > newCols {
			newCols = len(row.Cells)
		}
	}

	// Clear all existing text content
	for rowIdx, row := range existingTable.TableRows {
		if row == nil {
			continue
		}
		for colIdx, cell := range row.TableCells {
			if cell == nil || cell.Text == nil {
				continue
			}
			// Delete all text in the cell
			if hasTableCellContent(cell) {
				requests = append(requests, &slides.Request{
					DeleteText: &slides.DeleteTextRequest{
						ObjectId: tableObjectID,
						CellLocation: &slides.TableCellLocation{
							RowIndex:    int64(rowIdx),
							ColumnIndex: int64(colIdx),
						},
						TextRange: &slides.Range{
							Type: "ALL",
						},
					},
				})
			}
		}
	}

	// Adjust rows
	if newRows > existingRows {
		// Add rows - insert at the end of existing rows
		if existingRows > 0 {
			// Insert after the last existing row
			requests = append(requests, &slides.Request{
				InsertTableRows: &slides.InsertTableRowsRequest{
					TableObjectId: tableObjectID,
					CellLocation: &slides.TableCellLocation{
						RowIndex: int64(existingRows - 1), // Insert after the last existing row
					},
					InsertBelow: true,                          // Insert below the specified row
					Number:      int64(newRows - existingRows), // Number of rows to add
				},
			})
		}
	} else if newRows < existingRows {
		// Delete excess rows from the end
		for i := existingRows - 1; i >= newRows; i-- {
			requests = append(requests, &slides.Request{
				DeleteTableRow: &slides.DeleteTableRowRequest{
					TableObjectId: tableObjectID,
					CellLocation: &slides.TableCellLocation{
						RowIndex: int64(i),
					},
				},
			})
		}
	}

	// Adjust columns
	if newCols > existingCols {
		// Add columns - insert at the end of existing columns
		if existingCols > 0 {
			// Insert after the last existing column
			requests = append(requests, &slides.Request{
				InsertTableColumns: &slides.InsertTableColumnsRequest{
					TableObjectId: tableObjectID,
					CellLocation: &slides.TableCellLocation{
						ColumnIndex: int64(existingCols - 1), // Insert after the last existing column
					},
					InsertRight: true,                          // Insert to the right of the specified column
					Number:      int64(newCols - existingCols), // Number of columns to add
				},
			})
		}
	} else if newCols < existingCols {
		// Delete excess columns from the end
		for i := existingCols - 1; i >= newCols; i-- {
			requests = append(requests, &slides.Request{
				DeleteTableColumn: &slides.DeleteTableColumnRequest{
					TableObjectId: tableObjectID,
					CellLocation: &slides.TableCellLocation{
						ColumnIndex: int64(i),
					},
				},
			})
		}
	}

	return requests, nil
}

// hasTableCellContent checks if a table cell has any text content.
func hasTableCellContent(cell *slides.TableCell) bool {
	if cell == nil || cell.Text == nil {
		return false
	}
	for _, element := range cell.Text.TextElements {
		if element.TextRun != nil && strings.TrimSpace(element.TextRun.Content) != "" {
			return true
		}
	}
	return false
}

func (d *Deck) fillTableContentForActions(ctx context.Context, actions []*action) error {
	// Refresh to get the current slide structure with tables
	if err := d.refresh(ctx); err != nil {
		return fmt.Errorf("failed to refresh presentation: %w", err)
	}

	// Collect all table content requests across all slides that have table changes
	var allRequests []*slides.Request

	for _, action := range actions {
		if (action.actionType == actionTypeAppend || action.actionType == actionTypeUpdate) && len(action.slide.Tables) > 0 {
			var slideObjectID string
			if action.actionType == actionTypeAppend {
				// For appended slides, we need to find the correct slide
				if len(d.presentation.Slides) > action.index {
					slideObjectID = d.presentation.Slides[action.index].ObjectId
				}
			} else {
				// For updated slides
				if len(d.presentation.Slides) > action.index {
					slideObjectID = d.presentation.Slides[action.index].ObjectId
				}
			}

			if slideObjectID != "" {
				// Only fill content for slides that actually have table changes
				// This is determined by the handleTableUpdates logic
				requests, err := d.collectTableContentRequests(slideObjectID, action.slide.Tables)
				if err != nil {
					return fmt.Errorf("failed to collect table content requests for slide %d: %w", action.index, err)
				}
				allRequests = append(allRequests, requests...)
			}
		}
	}

	// Apply all table content updates in a single batch
	if len(allRequests) > 0 {
		if err := d.batchUpdate(ctx, allRequests); err != nil {
			return fmt.Errorf("failed to update all table content: %w", err)
		}
	}

	return nil
}

// createTableStructureRequest creates only the table structure without content.
func (d *Deck) createTableStructureRequest(slideObjectID string, table *Table, index int) ([]*slides.Request, error) {
	if len(table.Rows) == 0 {
		return nil, nil
	}

	tableObjectID := fmt.Sprintf("table-%s", uuid.New().String())

	// Calculate the number of rows and columns
	rows := int64(len(table.Rows))
	cols := int64(0)
	for _, row := range table.Rows {
		if int64(len(row.Cells)) > cols {
			cols = int64(len(row.Cells))
		}
	}

	if rows == 0 || cols == 0 {
		return nil, nil
	}

	// Create table request
	createTableReq := &slides.CreateTableRequest{
		ObjectId: tableObjectID,
		ElementProperties: &slides.PageElementProperties{
			PageObjectId: slideObjectID,
			Size: &slides.Size{
				Height: &slides.Dimension{
					Magnitude: float64(rows * 100000), // 100,000 EMU per row
					Unit:      "EMU",
				},
				Width: &slides.Dimension{
					Magnitude: float64(cols * 1000000), // 1,000,000 EMU per column
					Unit:      "EMU",
				},
			},
			Transform: &slides.AffineTransform{
				ScaleX:     1.0,
				ScaleY:     1.0,
				TranslateX: float64(index * 500000), // offset tables to avoid overlap
				TranslateY: float64(index * 100000),
				Unit:       "EMU",
			},
		},
		Rows:    rows,
		Columns: cols,
	}

	var requests []*slides.Request

	// Create table
	requests = append(requests, &slides.Request{
		CreateTable: createTableReq,
	})

	// Set description to mark as markdown-generated table
	requests = append(requests, &slides.Request{
		UpdatePageElementAltText: &slides.UpdatePageElementAltTextRequest{
			ObjectId:    tableObjectID,
			Description: descriptionTableFromMarkdown,
		},
	})

	return requests, nil
}

// collectTableContentRequests collects all table content requests for a slide.
func (d *Deck) collectTableContentRequests(slideObjectID string, tables []*Table) ([]*slides.Request, error) {
	if len(tables) == 0 {
		return nil, nil
	}

	var currentSlide *slides.Page
	for _, slide := range d.presentation.Slides {
		if slide.ObjectId == slideObjectID {
			currentSlide = slide
			break
		}
	}

	if currentSlide == nil {
		return nil, fmt.Errorf("slide not found: %s", slideObjectID)
	}

	// Find table elements in the slide
	var tableElements []*slides.PageElement
	for _, element := range currentSlide.PageElements {
		if element.Table != nil && element.Description == descriptionTableFromMarkdown {
			tableElements = append(tableElements, element)
		}
	}

	// If there are more existing tables than expected, we need to handle the mismatch
	// This can happen if previous tables weren't cleaned up properly
	if len(tableElements) > len(tables) {
		// Keep only the most recently created tables (assuming they are at the end)
		tableElements = tableElements[len(tableElements)-len(tables):]
	} else if len(tableElements) < len(tables) {
		return nil, fmt.Errorf("table count mismatch: expected %d tables but only found %d in slide", len(tables), len(tableElements))
	}

	// Collect content requests for each table
	var requests []*slides.Request
	for i, table := range tables {
		if i >= len(tableElements) {
			break
		}

		tableElement := tableElements[i]
		tableObjectID := tableElement.ObjectId

		// Check if the existing table already has content
		// If it does, skip content insertion to avoid duplicates
		if hasTableContent(tableElement.Table) {
			continue
		}

		tableReqs, err := d.createTableContentRequests(tableObjectID, table)
		if err != nil {
			return nil, fmt.Errorf("failed to create table content requests for table %d: %w", i, err)
		}
		requests = append(requests, tableReqs...)
	}

	return requests, nil
}

// createTableContentRequests creates requests to fill table content.
func (d *Deck) createTableContentRequests(tableObjectID string, table *Table) ([]*slides.Request, error) {
	var requests []*slides.Request

	// Fill table cells with content
	for rowIdx, row := range table.Rows {
		for colIdx, cell := range row.Cells {
			// Create text from fragments
			text := ""
			for _, fragment := range cell.Fragments {
				text += fragment.Value
			}

			if text == "" {
				continue
			}

			cellLocation := &slides.TableCellLocation{
				RowIndex:    int64(rowIdx),
				ColumnIndex: int64(colIdx),
			}

			// Insert text into cell
			requests = append(requests, &slides.Request{
				InsertText: &slides.InsertTextRequest{
					ObjectId:       tableObjectID,
					CellLocation:   cellLocation,
					Text:           text,
					InsertionIndex: 0,
				},
			})

			// Apply base text style from tableStyle (before fragment styles)
			textLength := int64(countString(text))
			if cellStyle := d.tableStyle.cellStyle(rowIdx, colIdx); cellStyle != nil && cellStyle.TextStyle != nil && textLength > 0 {
				req := buildTableCellTextStyleRequest(cellStyle.TextStyle)
				if req != nil {
					requests = append(requests, &slides.Request{
						UpdateTextStyle: &slides.UpdateTextStyleRequest{
							ObjectId:     tableObjectID,
							CellLocation: cellLocation,
							Style:        req.Style,
							TextRange: &slides.Range{
								Type:       "FIXED_RANGE",
								StartIndex: ptrInt64(0),
								EndIndex:   ptrInt64(textLength),
							},
							Fields: req.Fields,
						},
					})
				}
			}

			// Apply formatting if needed
			if len(cell.Fragments) > 0 {
				startIndex := int64(0)
				for _, fragment := range cell.Fragments {
					flen := countString(fragment.Value)
					if flen == 0 {
						continue
					}
					endIndex := startIndex + int64(flen)

					if r := d.getInlineStyleRequest(fragment); r != nil {
						requests = append(requests, &slides.Request{
							UpdateTextStyle: &slides.UpdateTextStyleRequest{
								ObjectId:     tableObjectID,
								CellLocation: cellLocation,
								Style:        r.Style,
								TextRange: &slides.Range{
									Type:       "FIXED_RANGE",
									StartIndex: ptrInt64(startIndex),
									EndIndex:   ptrInt64(endIndex),
								},
								Fields: r.Fields,
							},
						})
					}
					startIndex = endIndex
				}
			}

			// Set text alignment if specified
			if cell.Alignment != "" {
				requests = append(requests, &slides.Request{
					UpdateParagraphStyle: &slides.UpdateParagraphStyleRequest{
						ObjectId:     tableObjectID,
						CellLocation: cellLocation,
						Style: &slides.ParagraphStyle{
							Alignment: cell.Alignment,
						},
						Fields: "alignment",
						TextRange: &slides.Range{
							Type: "ALL",
						},
					},
				})
			}
		}
	}

	// Apply cell styles from tableStyle
	requests = append(requests, d.applyTableCellStyles(tableObjectID, table)...)

	// Apply border styles from tableStyle
	requests = append(requests, d.applyTableBorderStyles(tableObjectID, table)...)

	return requests, nil
}

// applyTableCellStyles applies cell styles from d.tableStyle.
func (d *Deck) applyTableCellStyles(tableObjectID string, table *Table) []*slides.Request {
	var requests []*slides.Request

	rows := len(table.Rows)
	if rows == 0 {
		return nil
	}

	cols := 0
	for _, row := range table.Rows {
		if len(row.Cells) > cols {
			cols = len(row.Cells)
		}
	}

	if cols == 0 {
		return nil
	}

	for rowIdx := 0; rowIdx < rows; rowIdx++ {
		for colIdx := 0; colIdx < cols; colIdx++ {
			cellStyle := d.tableStyle.cellStyle(rowIdx, colIdx)
			if cellStyle == nil {
				continue
			}

			// Apply background color
			if cellStyle.BackgroundFill != nil {
				requests = append(requests, &slides.Request{
					UpdateTableCellProperties: &slides.UpdateTableCellPropertiesRequest{
						ObjectId: tableObjectID,
						TableRange: &slides.TableRange{
							Location: &slides.TableCellLocation{
								RowIndex:    int64(rowIdx),
								ColumnIndex: int64(colIdx),
							},
							RowSpan:    1,
							ColumnSpan: 1,
						},
						TableCellProperties: &slides.TableCellProperties{
							TableCellBackgroundFill: cellStyle.BackgroundFill,
						},
						Fields: "tableCellBackgroundFill",
					},
				})
			}
		}
	}

	return requests
}

// applyTableBorderStyles applies border styles from d.tableStyle.BorderStyle.
func (d *Deck) applyTableBorderStyles(tableObjectID string, table *Table) []*slides.Request {
	if d.tableStyle == nil || d.tableStyle.BorderStyle == nil {
		return nil
	}

	var requests []*slides.Request
	bs := d.tableStyle.BorderStyle

	rows := len(table.Rows)
	if rows == 0 {
		return nil
	}

	cols := 0
	for _, row := range table.Rows {
		if len(row.Cells) > cols {
			cols = len(row.Cells)
		}
	}

	if cols == 0 {
		return nil
	}

	// Apply outer borders (top/bottom from OuterHorizontal, left/right from OuterVertical)
	if bs.OuterHorizontal != nil {
		outerH := prepareBorderProperties(bs.OuterHorizontal)
		// Top border of entire table
		requests = append(requests, &slides.Request{
			UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
				ObjectId:              tableObjectID,
				BorderPosition:        "TOP",
				TableBorderProperties: outerH,
				Fields:                buildBorderFields(outerH),
			},
		})
		// Bottom border of entire table
		requests = append(requests, &slides.Request{
			UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
				ObjectId:              tableObjectID,
				BorderPosition:        "BOTTOM",
				TableBorderProperties: outerH,
				Fields:                buildBorderFields(outerH),
			},
		})
	}

	if bs.OuterVertical != nil {
		outerV := prepareBorderProperties(bs.OuterVertical)
		// Left border of entire table
		requests = append(requests, &slides.Request{
			UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
				ObjectId:              tableObjectID,
				BorderPosition:        "LEFT",
				TableBorderProperties: outerV,
				Fields:                buildBorderFields(outerV),
			},
		})
		// Right border of entire table
		requests = append(requests, &slides.Request{
			UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
				ObjectId:              tableObjectID,
				BorderPosition:        "RIGHT",
				TableBorderProperties: outerV,
				Fields:                buildBorderFields(outerV),
			},
		})
	}

	// Apply inner borders per cell based on position
	for rowIdx := 0; rowIdx < rows; rowIdx++ {
		for colIdx := 0; colIdx < cols; colIdx++ {
			isHeaderRow := rowIdx == 0
			isFirstCol := colIdx == 0
			isLastRow := rowIdx == rows-1
			isLastCol := colIdx == cols-1

			tableRange := &slides.TableRange{
				Location: &slides.TableCellLocation{
					RowIndex:    int64(rowIdx),
					ColumnIndex: int64(colIdx),
				},
				RowSpan:    1,
				ColumnSpan: 1,
			}

			// Apply right border (skip outer right border)
			if !isLastCol {
				var srcProps *slides.TableBorderProperties
				if isHeaderRow {
					if isFirstCol {
						srcProps = bs.HeaderFirstColRight
					} else {
						srcProps = bs.HeaderOtherColRight
					}
				} else {
					if isFirstCol {
						srcProps = bs.DataFirstColRight
					} else {
						srcProps = bs.DataOtherColRight
					}
				}

				if srcProps != nil {
					borderProps := prepareBorderProperties(srcProps)
					requests = append(requests, &slides.Request{
						UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
							ObjectId:              tableObjectID,
							TableRange:            tableRange,
							BorderPosition:        "RIGHT",
							TableBorderProperties: borderProps,
							Fields:                buildBorderFields(borderProps),
						},
					})
				}
			}

			// Apply bottom border (skip outer bottom border)
			if !isLastRow {
				var srcProps *slides.TableBorderProperties
				if isHeaderRow {
					if isFirstCol {
						srcProps = bs.HeaderFirstColBottom
					} else {
						srcProps = bs.HeaderOtherColBottom
					}
				} else {
					if isFirstCol {
						srcProps = bs.DataFirstColBottom
					} else {
						srcProps = bs.DataOtherColBottom
					}
				}

				if srcProps != nil {
					borderProps := prepareBorderProperties(srcProps)
					requests = append(requests, &slides.Request{
						UpdateTableBorderProperties: &slides.UpdateTableBorderPropertiesRequest{
							ObjectId:              tableObjectID,
							TableRange:            tableRange,
							BorderPosition:        "BOTTOM",
							TableBorderProperties: borderProps,
							Fields:                buildBorderFields(borderProps),
						},
					})
				}
			}
		}
	}

	return requests
}

// buildBorderFields builds the fields string for UpdateTableBorderPropertiesRequest.
func buildBorderFields(props *slides.TableBorderProperties) string {
	if props == nil {
		return ""
	}

	var fields []string
	if props.TableBorderFill != nil {
		fields = append(fields, "tableBorderFill")
	}
	if props.Weight != nil {
		fields = append(fields, "weight")
	}
	if props.DashStyle != "" {
		fields = append(fields, "dashStyle")
	}

	if len(fields) == 0 {
		return "*"
	}
	return strings.Join(fields, ",")
}

// prepareBorderProperties creates a copy of TableBorderProperties with ForceSendFields
// set to ensure Alpha=0 (transparent) is properly sent to the API.
func prepareBorderProperties(props *slides.TableBorderProperties) *slides.TableBorderProperties {
	if props == nil {
		return nil
	}

	// Create a shallow copy
	result := &slides.TableBorderProperties{
		DashStyle: props.DashStyle,
		Weight:    props.Weight,
	}

	// Copy TableBorderFill with ForceSendFields for Alpha
	if props.TableBorderFill != nil && props.TableBorderFill.SolidFill != nil {
		sf := props.TableBorderFill.SolidFill
		result.TableBorderFill = &slides.TableBorderFill{
			SolidFill: &slides.SolidFill{
				Alpha:           sf.Alpha,
				Color:           sf.Color,
				ForceSendFields: []string{"Alpha"},
			},
		}
	}

	return result
}

// hasTableContent checks if a Google Slides table has any text content.
func hasTableContent(table *slides.Table) bool {
	if table == nil || len(table.TableRows) == 0 {
		return false
	}

	for _, row := range table.TableRows {
		if row == nil {
			continue
		}
		for _, cell := range row.TableCells {
			if cell == nil || cell.Text == nil {
				continue
			}
			// Check if the cell has any text elements with actual content
			for _, element := range cell.Text.TextElements {
				if element.TextRun != nil && element.TextRun.Content != "" {
					// Skip if it's just whitespace or newline
					trimmed := strings.TrimSpace(element.TextRun.Content)
					if trimmed != "" {
						return true
					}
				}
			}
		}
	}

	return false
}
