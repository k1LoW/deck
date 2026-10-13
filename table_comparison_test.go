package deck

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

func TestApplyUnchangedStyledTable(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "unchanged slides must not make API requests", http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)
	srv, err := slides.NewService(context.Background(),
		option.WithEndpoint(server.URL+"/"), option.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	d := &Deck{
		id:                 "presentation",
		srv:                srv,
		fresh:              true,
		defaultLayout:      "Body",
		defaultTitleLayout: "Body",
		tableStyle:         defaultTableStyle(),
		styles:             map[string]*slides.TextStyle{styleBold: {Bold: false, Italic: true}},
		logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		presentation: &slides.Presentation{
			Layouts: []*slides.Page{{
				ObjectId:         "layout",
				LayoutProperties: &slides.LayoutProperties{DisplayName: "Body"},
			}},
			Slides: []*slides.Page{{
				ObjectId: "slide",
				SlideProperties: &slides.SlideProperties{
					LayoutObjectId: "layout",
					NotesPage: &slides.Page{PageElements: []*slides.PageElement{{
						ObjectId: "notes",
						Shape:    &slides.Shape{Placeholder: &slides.Placeholder{Type: "BODY"}},
					}}},
				},
				PageElements: []*slides.PageElement{{
					ObjectId:    "table",
					Description: descriptionTableFromMarkdown,
					Table: &slides.Table{TableRows: []*slides.TableRow{{
						TableCells: []*slides.TableCell{{
							Text: &slides.TextContent{TextElements: []*slides.TextElement{
								{ParagraphMarker: &slides.ParagraphMarker{
									Style: &slides.ParagraphStyle{Alignment: "START"},
								}},
								{TextRun: &slides.TextRun{
									Content: "Header\n", Style: &slides.TextStyle{Bold: true},
								}},
							}},
						}},
					}}},
				}},
			}},
		},
	}
	target := Slides{{
		Layout: "Body",
		Tables: []*Table{{Rows: []*TableRow{{Cells: []*TableCell{{
			IsHeader:  true,
			Alignment: "START",
			Fragments: []*Fragment{{Value: "Header"}},
		}}}}}},
	}}
	if err := d.Apply(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("unchanged styled table made %d API requests", got)
	}
	if target[0].Tables[0].Rows[0].Cells[0].Fragments[0].Bold {
		t.Fatal("comparison changed the Markdown's explicit bold flag")
	}

	d.presentation.Slides = append(d.presentation.Slides, &slides.Page{
		ObjectId:        "untouched-slide",
		SlideProperties: d.presentation.Slides[0].SlideProperties,
		PageElements:    d.presentation.Slides[0].PageElements,
	})
	target = append(target, copySlide(target[0]))
	if err := d.ApplyPages(context.Background(), target, []int{1}); err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("partial apply changed an unselected styled table: %d API requests", got)
	}
}

func TestGenerateActionsWithTableStyles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		tableStyle   *TableStyle
		inlineStyles map[string]*slides.TextStyle
		markdown     []*Fragment
		rendered     []*Fragment
		alignment    string
		wantUpdate   bool
	}{
		{
			name:       "automatic header bold",
			tableStyle: defaultTableStyle(),
			markdown:   []*Fragment{{Value: "Header"}},
			rendered:   []*Fragment{{Value: "Header", Bold: true}},
		},
		{
			name:         "automatic bold does not apply named inline style",
			tableStyle:   defaultTableStyle(),
			inlineStyles: map[string]*slides.TextStyle{styleBold: {Bold: false, Italic: true}},
			markdown:     []*Fragment{{Value: "Header"}},
			rendered:     []*Fragment{{Value: "Header", Bold: true}},
		},
		{
			name:       "header without bold",
			tableStyle: &TableStyle{HeaderFirstCol: &TableCellStyle{TextStyle: &slides.TextStyle{}}},
			markdown:   []*Fragment{{Value: "Header"}},
			rendered:   []*Fragment{{Value: "Header"}},
		},
		{
			name: "custom table italic and code style",
			tableStyle: &TableStyle{HeaderFirstCol: &TableCellStyle{
				TextStyle: &slides.TextStyle{Italic: true, FontFamily: defaultCodeFontFamily},
			}},
			markdown: []*Fragment{{Value: "Header"}},
			rendered: []*Fragment{{Value: "Header", Italic: true, Code: true}},
		},
		{
			name:       "explicit italic preserves base bold",
			tableStyle: defaultTableStyle(),
			markdown:   []*Fragment{{Value: "Header", Italic: true}},
			rendered:   []*Fragment{{Value: "Header", Bold: true, Italic: true}},
		},
		{
			name:         "named bold can unset base bold",
			tableStyle:   defaultTableStyle(),
			inlineStyles: map[string]*slides.TextStyle{styleBold: {Bold: false, Italic: true}},
			markdown:     []*Fragment{{Value: "Header", Bold: true}},
			rendered:     []*Fragment{{Value: "Header", Italic: true}},
		},
		{
			name:         "named style is compared by its rendered result",
			tableStyle:   defaultTableStyle(),
			inlineStyles: map[string]*slides.TextStyle{"custom": {Bold: false, Italic: true}},
			markdown:     []*Fragment{{Value: "Header", StyleName: "custom"}},
			rendered:     []*Fragment{{Value: "Header", Italic: true}},
		},
		{
			name: "template alignment overrides markdown alignment",
			tableStyle: &TableStyle{HeaderFirstCol: &TableCellStyle{
				ParagraphStyle: &slides.ParagraphStyle{Alignment: "CENTER"},
			}},
			markdown:  []*Fragment{{Value: "Header"}},
			rendered:  []*Fragment{{Value: "Header"}},
			alignment: "CENTER",
		},
		{
			name:       "API merges fragments with identical rendered styles",
			tableStyle: defaultTableStyle(),
			markdown:   []*Fragment{{Value: "Head"}, {Value: "er", Bold: true}},
			rendered:   []*Fragment{{Value: "Header", Bold: true}},
		},
		{
			name:       "explicit bold added",
			tableStyle: &TableStyle{},
			markdown:   []*Fragment{{Value: "Header", Bold: true}},
			rendered:   []*Fragment{{Value: "Header"}},
			wantUpdate: true,
		},
		{
			name:       "explicit bold removed",
			tableStyle: &TableStyle{},
			markdown:   []*Fragment{{Value: "Header"}},
			rendered:   []*Fragment{{Value: "Header", Bold: true}},
			wantUpdate: true,
		},
		{
			name:         "named bold removed restores base style",
			tableStyle:   defaultTableStyle(),
			inlineStyles: map[string]*slides.TextStyle{styleBold: {Bold: false, Italic: true}},
			markdown:     []*Fragment{{Value: "Header"}},
			rendered:     []*Fragment{{Value: "Header", Italic: true}},
			wantUpdate:   true,
		},
		{
			name:       "text changed",
			tableStyle: defaultTableStyle(),
			markdown:   []*Fragment{{Value: "New"}},
			rendered:   []*Fragment{{Value: "Old", Bold: true}},
			wantUpdate: true,
		},
		{
			name:       "alignment changed",
			tableStyle: defaultTableStyle(),
			markdown:   []*Fragment{{Value: "Header"}},
			rendered:   []*Fragment{{Value: "Header", Bold: true}},
			alignment:  "END",
			wantUpdate: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &Deck{tableStyle: tt.tableStyle, styles: tt.inlineStyles}
			after := Slides{{
				Layout: "Body",
				Tables: []*Table{{Rows: []*TableRow{{Cells: []*TableCell{{
					IsHeader: true, Alignment: "START", Fragments: tt.markdown,
				}}}}}},
			}}
			original := copySlides(after)
			before := copySlides(after)
			beforeCell := before[0].Tables[0].Rows[0].Cells[0]
			beforeCell.Fragments = tt.rendered
			if tt.alignment != "" {
				beforeCell.Alignment = tt.alignment
			}
			actions, err := generateActionsWithTableStyles(before, after, func(_ int, tables []*Table) []*Table {
				return d.renderTablesForComparison(tables)
			})
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantUpdate {
				if len(actions) != 1 || actions[0].actionType != actionTypeUpdate {
					t.Fatalf("expected one update, got %v", toActionLogs(actions))
				}
				if diff := cmp.Diff(after[0].Tables, actions[0].slide.Tables); diff != "" {
					t.Fatalf("action must retain explicit Markdown styles (-want +got):\n%s", diff)
				}
			} else if len(actions) != 0 {
				t.Fatalf("unchanged rendered content produced actions: %v", toActionLogs(actions))
			}
			if diff := cmp.Diff(original, after, cmp.AllowUnexported(Slide{})); diff != "" {
				t.Fatalf("comparison mutated input (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGenerateActionsWithStyledTableSlideChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		before []string
		after  []string
		want   []actionType
	}{
		{
			name:   "reorder",
			before: []string{"A", "B", "C"},
			after:  []string{"B", "C", "A"},
			want:   []actionType{actionTypeMove, actionTypeMove},
		},
		{
			name:   "insert",
			before: []string{"A", "C"},
			after:  []string{"A", "B", "C"},
			want:   []actionType{actionTypeAppend, actionTypeMove},
		},
		{
			name:   "delete",
			before: []string{"A", "B", "C"},
			after:  []string{"A", "C"},
			want:   []actionType{actionTypeDelete},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			makeSlides := func(values []string, bold bool) Slides {
				result := make(Slides, len(values))
				for i, value := range values {
					result[i] = &Slide{
						Layout: "Body",
						Tables: []*Table{{Rows: []*TableRow{{Cells: []*TableCell{{
							IsHeader: true, Alignment: "START",
							Fragments: []*Fragment{{Value: value, Bold: bold}},
						}}}}}},
					}
				}
				return result
			}
			d := &Deck{tableStyle: defaultTableStyle()}
			actions, err := generateActionsWithTableStyles(makeSlides(tt.before, true), makeSlides(tt.after, false),
				func(_ int, tables []*Table) []*Table { return d.renderTablesForComparison(tables) })
			if err != nil {
				t.Fatal(err)
			}
			var got []actionType
			for _, action := range actions {
				got = append(got, action.actionType)
				if action.actionType == actionTypeAppend && action.slide.Tables[0].Rows[0].Cells[0].Fragments[0].Bold {
					t.Fatal("appended slide lost the original Markdown styles")
				}
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatalf("unexpected actions (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandleUnchangedStyledTable(t *testing.T) {
	t.Parallel()

	d := &Deck{tableStyle: defaultTableStyle()}
	table := &Table{Rows: []*TableRow{{Cells: []*TableCell{{
		IsHeader: true, Alignment: "START", Fragments: []*Fragment{{Value: "Header"}},
	}}}}}
	current := []*slides.PageElement{{
		ObjectId:    "table",
		Description: descriptionTableFromMarkdown,
		Table: &slides.Table{TableRows: []*slides.TableRow{{
			TableCells: []*slides.TableCell{{
				Text: &slides.TextContent{TextElements: []*slides.TextElement{
					{ParagraphMarker: &slides.ParagraphMarker{
						Style: &slides.ParagraphStyle{Alignment: "START"},
					}},
					{TextRun: &slides.TextRun{Content: "Header\n", Style: &slides.TextStyle{Bold: true}}},
				}},
			}},
		}}},
	}}
	requests, err := d.handleTableUpdates("slide", []*Table{table}, current)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 0 {
		t.Fatalf("unchanged table must not be cleared during other slide updates: %v", requests)
	}
}
