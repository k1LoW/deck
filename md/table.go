package md

import (
	"github.com/k1LoW/deck"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// parseTable parses an east.Table node and converts it to our Table structure.
func parseTable(tableNode *east.Table, baseDir string, b []byte, breaks bool) (*deck.Table, error) {
	table := &deck.Table{
		Rows: []*deck.TableRow{},
	}

	for child := tableNode.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *east.TableHeader:
			// Parse table header row
			row, err := parseTableRow(v, baseDir, b, breaks, true)
			if err != nil {
				return nil, err
			}
			table.Rows = append(table.Rows, row)

		case *east.TableRow:
			// Parse regular table row
			row, err := parseTableRow(v, baseDir, b, breaks, false)
			if err != nil {
				return nil, err
			}
			table.Rows = append(table.Rows, row)
		}
	}

	return table, nil
}

// parseTableRow parses a table row (header or regular) and extracts cells.
func parseTableRow(rowNode ast.Node, baseDir string, b []byte, breaks, isHeader bool) (*deck.TableRow, error) {
	row := &deck.TableRow{
		Cells: []*deck.TableCell{},
	}

	for child := rowNode.FirstChild(); child != nil; child = child.NextSibling() {
		if cellNode, ok := child.(*east.TableCell); ok {
			cell, err := parseTableCell(cellNode, baseDir, b, breaks, isHeader)
			if err != nil {
				return nil, err
			}
			row.Cells = append(row.Cells, cell)
		}
	}

	return row, nil
}

// parseTableCell parses a table cell and extracts its content and alignment.
func parseTableCell(cellNode *east.TableCell, baseDir string, b []byte, breaks, isHeader bool) (*deck.TableCell, error) {
	cell := &deck.TableCell{
		Fragments: []*deck.Fragment{},
		IsHeader:  isHeader,
		Alignment: "START", // Default alignment for LTR text
	}
	// When the case of east.AlignNone, we can use an empty string as an Alignment, but even if it is an
	// empty string, "START" is returned once it is reflected in the API, so "START" is set here by
	// default for comparison.
	switch cellNode.Alignment {
	case east.AlignCenter:
		cell.Alignment = "CENTER"
	case east.AlignRight:
		cell.Alignment = "END"
	}

	seedFragment := deck.Fragment{}
	if isHeader {
		seedFragment.Bold = true
	}
	// Parse cell content to fragments
	frags, _, err := toFragments(baseDir, b, cellNode, seedFragment)
	if err != nil {
		return nil, err
	}
	// Convert to deck fragments
	cell.Fragments = toDeckFragments(frags, breaks)

	return cell, nil
}
