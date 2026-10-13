package deck

import (
	"strings"
	"testing"

	"google.golang.org/api/slides/v1"
)

func TestTableTextComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fragments []*Fragment
		apiRuns   []*slides.TextRun
		text      string
	}{
		{
			name:      "trailing user newline",
			fragments: []*Fragment{{Value: "Header\n"}},
			apiRuns:   []*slides.TextRun{{Content: "Header\n\n"}},
			text:      "Header\n",
		},
		{
			name:      "separate API terminator",
			fragments: []*Fragment{{Value: "Header\n", Bold: true}},
			apiRuns: []*slides.TextRun{
				{Content: "Header\n", Style: &slides.TextStyle{Bold: true}},
				{Content: "\n", Style: &slides.TextStyle{Italic: true}},
			},
			text: "Header\n",
		},
		{
			name:      "multiple trailing user newlines",
			fragments: []*Fragment{{Value: "Header\n\n"}},
			apiRuns:   []*slides.TextRun{{Content: "Header\n\n\n"}},
			text:      "Header\n\n",
		},
		{
			name:      "separate newline fragment",
			fragments: []*Fragment{{Value: "A"}, {Value: "\n"}, {Value: "B"}},
			apiRuns:   []*slides.TextRun{{Content: "A\n"}, {Content: "B\n"}},
			text:      "A\nB",
		},
		{
			name:      "API soft line break",
			fragments: []*Fragment{{Value: "A"}, {Value: "\n"}, {Value: "B"}},
			apiRuns:   []*slides.TextRun{{Content: "A\vB\n"}},
			text:      "A\nB",
		},
		{
			name:      "styled paragraph boundary",
			fragments: []*Fragment{{Value: "A\n", Bold: true}, {Value: "B", Italic: true}},
			apiRuns: []*slides.TextRun{
				{Content: "A\n", Style: &slides.TextStyle{Bold: true}},
				{Content: "B\n", Style: &slides.TextStyle{Italic: true}},
			},
			text: "A\nB",
		},
		{
			name:      "leading newline",
			fragments: []*Fragment{{Value: "\n"}, {Value: "B"}},
			apiRuns:   []*slides.TextRun{{Content: "\n"}, {Content: "B\n"}},
			text:      "\nB",
		},
		{
			name:      "newline-only cell",
			fragments: []*Fragment{{Value: "\n"}},
			apiRuns:   []*slides.TextRun{{Content: "\n\n"}},
			text:      "\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			table := &Table{Rows: []*TableRow{{Cells: []*TableCell{{
				IsHeader: true, Alignment: "START", Fragments: tt.fragments,
			}}}}}
			d := &Deck{tableStyle: &TableStyle{}}
			rendered := d.renderTable(table)
			var text strings.Builder
			for _, fragment := range rendered.content.Rows[0].Cells[0].Fragments {
				text.WriteString(fragment.Value)
			}
			if got := text.String(); got != tt.text {
				t.Fatalf("render comparison lost user text: got %q, want %q", got, tt.text)
			}
			elements := []*slides.TextElement{{
				ParagraphMarker: &slides.ParagraphMarker{Style: &slides.ParagraphStyle{Alignment: "START"}},
			}}
			for _, run := range tt.apiRuns {
				elements = append(elements, &slides.TextElement{TextRun: run})
			}
			current := convertSlidesToTable(&slides.Table{TableRows: []*slides.TableRow{{
				TableCells: []*slides.TableCell{{Text: &slides.TextContent{TextElements: elements}}},
			}}})
			if !tablesEqual([]*Table{current}, []*Table{rendered.content}) {
				t.Fatal("unchanged text with line breaks must not trigger a table update")
			}

			withoutBreaks := copySlide(&Slide{Tables: []*Table{table}}).Tables[0]
			for _, fragment := range withoutBreaks.Rows[0].Cells[0].Fragments {
				fragment.Value = strings.ReplaceAll(fragment.Value, "\n", "")
			}
			if tablesEqual([]*Table{current}, []*Table{d.renderTable(withoutBreaks).content}) {
				t.Fatal("removing a user line break must trigger a table update")
			}
		})
	}
}

func TestTableWhitespaceContent(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name    string
		content string
		want    bool
	}{
		{name: "API terminator only", content: "\n"},
		{name: "user newline", content: "\n\n", want: true},
		{name: "user space", content: " \n", want: true},
		{name: "text", content: "A\n", want: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cell := &slides.TableCell{Text: &slides.TextContent{
				TextElements: []*slides.TextElement{{TextRun: &slides.TextRun{Content: tt.content}}},
			}}
			if got := hasTableCellContent(cell); got != tt.want {
				t.Fatalf("cell content: got %v, want %v", got, tt.want)
			}
			table := &slides.Table{TableRows: []*slides.TableRow{{TableCells: []*slides.TableCell{cell}}}}
			if got := hasTableContent(table); got != tt.want {
				t.Fatalf("table content: got %v, want %v", got, tt.want)
			}
		})
	}
}
