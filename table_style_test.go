package deck

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/slides/v1"
)

func TestCellStyle(t *testing.T) {
	t.Parallel()
	ts := defaultTableStyle()

	tests := []struct {
		name   string
		rowIdx int
		colIdx int
		want   *TableCellStyle
	}{
		{
			name:   "header first column",
			rowIdx: 0,
			colIdx: 0,
			want:   ts.HeaderFirstCol,
		},
		{
			name:   "header second column",
			rowIdx: 0,
			colIdx: 1,
			want:   ts.HeaderOtherCols,
		},
		{
			name:   "header third column",
			rowIdx: 0,
			colIdx: 2,
			want:   ts.HeaderOtherCols,
		},
		{
			name:   "data first column row 1",
			rowIdx: 1,
			colIdx: 0,
			want:   ts.DataFirstCol,
		},
		{
			name:   "data first column row 2",
			rowIdx: 2,
			colIdx: 0,
			want:   ts.DataFirstCol,
		},
		{
			name:   "data other column row 1 col 1",
			rowIdx: 1,
			colIdx: 1,
			want:   ts.DataOtherCols,
		},
		{
			name:   "data other column row 2 col 2",
			rowIdx: 2,
			colIdx: 2,
			want:   ts.DataOtherCols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ts.cellStyle(tt.rowIdx, tt.colIdx)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCellStyleNilTableStyle(t *testing.T) {
	t.Parallel()
	var ts *TableStyle
	got := ts.cellStyle(0, 0)
	if got != nil {
		t.Error("cellStyle on nil TableStyle should return nil")
	}
}

func TestExtractTableStyleFromLayout_Valid2x2(t *testing.T) {
	t.Parallel()
	// Create a 2x2 table with different styles for each cell
	table := &slides.Table{
		TableRows: []*slides.TableRow{
			{
				TableCells: []*slides.TableCell{
					{
						TableCellProperties: &slides.TableCellProperties{
							TableCellBackgroundFill: &slides.TableCellBackgroundFill{
								SolidFill: &slides.SolidFill{
									Color: &slides.OpaqueColor{
										RgbColor: &slides.RgbColor{Red: 1.0, Green: 0.0, Blue: 0.0},
									},
								},
							},
						},
						Text: &slides.TextContent{
							TextElements: []*slides.TextElement{
								{
									TextRun: &slides.TextRun{
										Style: &slides.TextStyle{Bold: true},
									},
								},
							},
						},
					},
					{
						TableCellProperties: &slides.TableCellProperties{
							TableCellBackgroundFill: &slides.TableCellBackgroundFill{
								SolidFill: &slides.SolidFill{
									Color: &slides.OpaqueColor{
										RgbColor: &slides.RgbColor{Red: 0.0, Green: 1.0, Blue: 0.0},
									},
								},
							},
						},
						Text: &slides.TextContent{
							TextElements: []*slides.TextElement{
								{
									TextRun: &slides.TextRun{
										Style: &slides.TextStyle{Italic: true},
									},
								},
							},
						},
					},
				},
			},
			{
				TableCells: []*slides.TableCell{
					{
						TableCellProperties: &slides.TableCellProperties{
							TableCellBackgroundFill: &slides.TableCellBackgroundFill{
								SolidFill: &slides.SolidFill{
									Color: &slides.OpaqueColor{
										RgbColor: &slides.RgbColor{Red: 0.0, Green: 0.0, Blue: 1.0},
									},
								},
							},
						},
					},
					{
						TableCellProperties: &slides.TableCellProperties{
							TableCellBackgroundFill: &slides.TableCellBackgroundFill{
								SolidFill: &slides.SolidFill{
									Color: &slides.OpaqueColor{
										RgbColor: &slides.RgbColor{Red: 1.0, Green: 1.0, Blue: 0.0},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	got := extractTableStyleFromLayout(table)
	if got == nil {
		t.Fatal("extractTableStyleFromLayout returned nil for valid 2x2 table")
	}

	want := &TableStyle{
		HeaderFirstCol: &TableCellStyle{
			BackgroundFill: &slides.TableCellBackgroundFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{Red: 1.0, Green: 0.0, Blue: 0.0},
					},
				},
			},
			TextStyle: &slides.TextStyle{Bold: true},
		},
		HeaderOtherCols: &TableCellStyle{
			BackgroundFill: &slides.TableCellBackgroundFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{Red: 0.0, Green: 1.0, Blue: 0.0},
					},
				},
			},
			TextStyle: &slides.TextStyle{Italic: true},
		},
		DataFirstCol: &TableCellStyle{
			BackgroundFill: &slides.TableCellBackgroundFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{Red: 0.0, Green: 0.0, Blue: 1.0},
					},
				},
			},
		},
		DataOtherCols: &TableCellStyle{
			BackgroundFill: &slides.TableCellBackgroundFill{
				SolidFill: &slides.SolidFill{
					Color: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{Red: 1.0, Green: 1.0, Blue: 0.0},
					},
				},
			},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("extractTableStyleFromLayout() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractTableStyleFromLayout_Non2x2(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		table *slides.Table
	}{
		{
			name: "1x2 table (1 row)",
			table: &slides.Table{
				TableRows: []*slides.TableRow{
					{
						TableCells: []*slides.TableCell{{}, {}},
					},
				},
			},
		},
		{
			name: "3x2 table (3 rows)",
			table: &slides.Table{
				TableRows: []*slides.TableRow{
					{TableCells: []*slides.TableCell{{}, {}}},
					{TableCells: []*slides.TableCell{{}, {}}},
					{TableCells: []*slides.TableCell{{}, {}}},
				},
			},
		},
		{
			name: "2x1 table (1 column)",
			table: &slides.Table{
				TableRows: []*slides.TableRow{
					{TableCells: []*slides.TableCell{{}}},
					{TableCells: []*slides.TableCell{{}}},
				},
			},
		},
		{
			name: "2x3 table (3 columns)",
			table: &slides.Table{
				TableRows: []*slides.TableRow{
					{TableCells: []*slides.TableCell{{}, {}, {}}},
					{TableCells: []*slides.TableCell{{}, {}, {}}},
				},
			},
		},
		{
			name: "empty table",
			table: &slides.Table{
				TableRows: []*slides.TableRow{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ts := extractTableStyleFromLayout(tt.table)
			if ts != nil {
				t.Errorf("extractTableStyleFromLayout should return nil for %s", tt.name)
			}
		})
	}
}

func TestBuildTableCellTextStyleRequest(t *testing.T) {
	t.Parallel()
	t.Run("nil input", func(t *testing.T) {
		t.Parallel()
		req := buildTableCellTextStyleRequest(nil)
		if req != nil {
			t.Error("should return nil for nil input")
		}
	})

	t.Run("empty style", func(t *testing.T) {
		t.Parallel()
		req := buildTableCellTextStyleRequest(&slides.TextStyle{})
		if req == nil {
			t.Fatal("should not return nil")
		}
		// Boolean fields are always included
		if req.Style.Bold {
			t.Error("Bold should be false")
		}
		for _, f := range []string{"bold", "italic", "underline", "strikethrough"} {
			if !strings.Contains(req.Fields, f) {
				t.Errorf("Fields should contain %q", f)
			}
		}
	})

	t.Run("bold only", func(t *testing.T) {
		t.Parallel()
		req := buildTableCellTextStyleRequest(&slides.TextStyle{Bold: true})
		if req == nil {
			t.Fatal("should not return nil")
		}
		if !req.Style.Bold {
			t.Error("Bold should be true")
		}
		if !strings.Contains(req.Fields, "bold") {
			t.Error("Fields should contain bold")
		}
	})

	t.Run("multiple properties", func(t *testing.T) {
		t.Parallel()
		req := buildTableCellTextStyleRequest(&slides.TextStyle{
			Bold:       true,
			Italic:     true,
			FontFamily: "Arial",
			ForegroundColor: &slides.OptionalColor{
				OpaqueColor: &slides.OpaqueColor{
					RgbColor: &slides.RgbColor{Red: 1.0},
				},
			},
		})
		if req == nil {
			t.Fatal("should not return nil")
		}
		if !req.Style.Bold {
			t.Error("Bold should be true")
		}
		if !req.Style.Italic {
			t.Error("Italic should be true")
		}
		if req.Style.FontFamily != "Arial" {
			t.Errorf("FontFamily = %q, want %q", req.Style.FontFamily, "Arial")
		}
		if req.Style.ForegroundColor == nil {
			t.Error("ForegroundColor should not be nil")
		}
	})

	t.Run("all properties", func(t *testing.T) {
		t.Parallel()
		req := buildTableCellTextStyleRequest(&slides.TextStyle{
			Bold:          true,
			Italic:        true,
			Underline:     true,
			Strikethrough: true,
			FontFamily:    "Roboto",
			ForegroundColor: &slides.OptionalColor{
				OpaqueColor: &slides.OpaqueColor{
					RgbColor: &slides.RgbColor{Red: 0.5},
				},
			},
			BackgroundColor: &slides.OptionalColor{
				OpaqueColor: &slides.OpaqueColor{
					RgbColor: &slides.RgbColor{Green: 0.5},
				},
			},
			FontSize: &slides.Dimension{
				Magnitude: 12,
				Unit:      "PT",
			},
			BaselineOffset: "SUPERSCRIPT",
		})
		if req == nil {
			t.Fatal("should not return nil")
		}
		// Check that all fields are included
		expectedFields := []string{"bold", "italic", "underline", "strikethrough", "fontFamily", "foregroundColor", "backgroundColor", "fontSize", "baselineOffset"}
		for _, f := range expectedFields {
			if !strings.Contains(req.Fields, f) {
				t.Errorf("Fields should contain %q, got %q", f, req.Fields)
			}
		}
	})
}

func TestExtractCellStyle(t *testing.T) {
	t.Parallel()
	t.Run("cell with background and text style", func(t *testing.T) {
		t.Parallel()
		cell := &slides.TableCell{
			TableCellProperties: &slides.TableCellProperties{
				TableCellBackgroundFill: &slides.TableCellBackgroundFill{
					SolidFill: &slides.SolidFill{
						Color: &slides.OpaqueColor{
							RgbColor: &slides.RgbColor{Red: 0.5, Green: 0.5, Blue: 0.5},
						},
					},
				},
			},
			Text: &slides.TextContent{
				TextElements: []*slides.TextElement{
					{
						TextRun: &slides.TextRun{
							Style: &slides.TextStyle{
								Bold:       true,
								FontFamily: "Arial",
							},
						},
					},
				},
			},
		}

		style := extractCellStyle(cell)
		if style == nil {
			t.Fatal("extractCellStyle returned nil")
		}
		if style.BackgroundFill == nil {
			t.Error("BackgroundFill is nil")
		}
		if style.TextStyle == nil {
			t.Error("TextStyle is nil")
		} else {
			if !style.TextStyle.Bold {
				t.Error("TextStyle.Bold should be true")
			}
			if style.TextStyle.FontFamily != "Arial" {
				t.Errorf("TextStyle.FontFamily = %q, want %q", style.TextStyle.FontFamily, "Arial")
			}
		}
	})

	t.Run("cell with only background", func(t *testing.T) {
		t.Parallel()
		cell := &slides.TableCell{
			TableCellProperties: &slides.TableCellProperties{
				TableCellBackgroundFill: &slides.TableCellBackgroundFill{
					SolidFill: &slides.SolidFill{
						Color: &slides.OpaqueColor{
							RgbColor: &slides.RgbColor{Red: 1.0, Green: 0.0, Blue: 0.0},
						},
					},
				},
			},
		}

		style := extractCellStyle(cell)
		if style == nil {
			t.Fatal("extractCellStyle returned nil")
		}
		if style.BackgroundFill == nil {
			t.Error("BackgroundFill is nil")
		}
		if style.TextStyle != nil {
			t.Error("TextStyle should be nil")
		}
	})

	t.Run("cell with only text style", func(t *testing.T) {
		t.Parallel()
		cell := &slides.TableCell{
			Text: &slides.TextContent{
				TextElements: []*slides.TextElement{
					{
						TextRun: &slides.TextRun{
							Style: &slides.TextStyle{Italic: true},
						},
					},
				},
			},
		}

		style := extractCellStyle(cell)
		if style == nil {
			t.Fatal("extractCellStyle returned nil")
		}
		if style.BackgroundFill != nil {
			t.Error("BackgroundFill should be nil")
		}
		if style.TextStyle == nil {
			t.Error("TextStyle is nil")
		} else if !style.TextStyle.Italic {
			t.Error("TextStyle.Italic should be true")
		}
	})

	t.Run("empty cell", func(t *testing.T) {
		t.Parallel()
		cell := &slides.TableCell{}

		style := extractCellStyle(cell)
		if style == nil {
			t.Fatal("extractCellStyle returned nil")
		}
		if style.BackgroundFill != nil {
			t.Error("BackgroundFill should be nil")
		}
		if style.TextStyle != nil {
			t.Error("TextStyle should be nil")
		}
	})

	t.Run("cell with multiple text elements (first one used)", func(t *testing.T) {
		t.Parallel()
		cell := &slides.TableCell{
			Text: &slides.TextContent{
				TextElements: []*slides.TextElement{
					{
						TextRun: &slides.TextRun{
							Style: &slides.TextStyle{Bold: true},
						},
					},
					{
						TextRun: &slides.TextRun{
							Style: &slides.TextStyle{Italic: true},
						},
					},
				},
			},
		}

		style := extractCellStyle(cell)
		if style == nil {
			t.Fatal("extractCellStyle returned nil")
		}
		if style.TextStyle == nil {
			t.Fatal("TextStyle is nil")
		}
		// Should use the first TextRun's style
		if !style.TextStyle.Bold {
			t.Error("TextStyle.Bold should be true (from first TextRun)")
		}
		if style.TextStyle.Italic {
			t.Error("TextStyle.Italic should be false (second TextRun should be ignored)")
		}
	})
}
