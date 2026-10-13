package deck

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/slides/v1"
)

func TestApplyStyleSnapshot(t *testing.T) {
	t.Parallel()

	backend := styleSnapshotPresentation()
	var d *Deck
	var reads, writes atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			reads.Add(1)
			if err := json.NewEncoder(w).Encode(backend); err != nil {
				t.Error(err)
			}
			return
		}
		var batch slides.BatchUpdatePresentationRequest
		if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		cell := backend.Slides[0].PageElements[0].Table.TableRows[0].TableCells[0]
		for _, request := range batch.Requests {
			switch {
			case request.DeleteText != nil && request.DeleteText.ObjectId == "table":
				cell.Text.TextElements = nil
			case request.InsertText != nil && request.InsertText.ObjectId == "table":
				cell.Text.TextElements = []*slides.TextElement{
					{ParagraphMarker: &slides.ParagraphMarker{Style: &slides.ParagraphStyle{Alignment: "START"}}},
					{TextRun: &slides.TextRun{Content: request.InsertText.Text + "\n"}},
				}
			case request.UpdateTextStyle != nil && request.UpdateTextStyle.ObjectId == "table":
				run := cell.Text.TextElements[1].TextRun
				run.Style = mergeStyles(run.Style, request.UpdateTextStyle.Style, request.UpdateTextStyle.Fields)
			}
		}
		if writes.Add(1) == 1 {
			// Change both the already fetched layout and the next API response mid-apply.
			currentStyle := d.presentation.Layouts[1].PageElements[0].Shape.Text.TextElements[0].TextRun.Style
			currentStyle.ForegroundColor.OpaqueColor.RgbColor.Red = 0.7
			currentCell := d.presentation.Layouts[1].PageElements[1].Table.TableRows[0].TableCells[0]
			currentCell.Text.TextElements[0].TextRun.Style.FontSize.Magnitude = 25
			currentCell.TableCellProperties.TableCellBackgroundFill.SolidFill.Color.RgbColor.Red = 0.7
			currentShape := d.presentation.Layouts[1].PageElements[0].Shape.ShapeProperties
			currentShape.ShapeBackgroundFill.SolidFill.Color.RgbColor.Red = 0.7
			currentBorder := d.presentation.Layouts[1].PageElements[1].Table.HorizontalBorderRows[0].TableBorderCells[0]
			currentBorder.TableBorderProperties.Weight.Magnitude = 7
			nextStyle := backend.Layouts[1].PageElements[0].Shape.Text.TextElements[0].TextRun.Style
			nextStyle.ForegroundColor.OpaqueColor.RgbColor.Red = 0.8
			nextCell := backend.Layouts[1].PageElements[1].Table.TableRows[0].TableCells[0]
			nextCell.Text.TextElements[0].TextRun.Style.FontSize.Magnitude = 30
			nextCell.TableCellProperties.TableCellBackgroundFill.SolidFill.Color.RgbColor.Red = 0.8
			nextShape := backend.Layouts[1].PageElements[0].Shape.ShapeProperties
			nextShape.ShapeBackgroundFill.SolidFill.Color.RgbColor.Red = 0.8
			nextBorder := backend.Layouts[1].PageElements[1].Table.HorizontalBorderRows[0].TableBorderCells[0]
			nextBorder.TableBorderProperties.Weight.Magnitude = 3
		}
		if _, err := io.WriteString(w, "{}"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	srv, err := slides.NewService(context.Background(),
		option.WithEndpoint(server.URL+"/"), option.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	d = &Deck{
		id: "presentation", srv: srv,
		styles: map[string]*slides.TextStyle{}, shapes: map[string]*slides.ShapeProperties{},
		tableStyle: defaultTableStyle(), logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	target := Slides{{
		Layout: "Body",
		Tables: []*Table{{Rows: []*TableRow{{Cells: []*TableCell{{
			IsHeader: true, Alignment: "START",
			Fragments: []*Fragment{{Value: "Changed", Bold: true}},
		}}}}}},
	}}
	if err := d.Apply(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if got := d.styles[styleBold].ForegroundColor.OpaqueColor.RgbColor.Red; got != 0.2 {
		t.Fatalf("style changed during apply: got %v, want 0.2", got)
	}
	if got := d.tableStyle.HeaderFirstCol.TextStyle.FontSize.Magnitude; got != 14 {
		t.Fatalf("table font changed during apply: got %v, want 14", got)
	}
	if got := d.tableStyle.HeaderFirstCol.BackgroundFill.SolidFill.Color.RgbColor.Red; got != 0.2 {
		t.Fatalf("table background changed during apply: got %v, want 0.2", got)
	}
	if got := d.shapes[styleBold].ShapeBackgroundFill.SolidFill.Color.RgbColor.Red; got != 0.2 {
		t.Fatalf("shape background changed during apply: got %v, want 0.2", got)
	}
	if got := d.tableStyle.BorderStyle.OuterHorizontal.Weight.Magnitude; got != 1 {
		t.Fatalf("border weight changed during apply: got %v, want 1", got)
	}
	uploaded := backend.Slides[0].PageElements[0].Table.TableRows[0].TableCells[0].Text.TextElements[1].TextRun.Style
	if got := uploaded.FontSize.Magnitude; got != 14 {
		t.Fatalf("upload did not use the frozen font: got %v, want 14", got)
	}
	if got := uploaded.ForegroundColor.OpaqueColor.RgbColor.Red; got != 0.2 {
		t.Fatalf("upload did not use the frozen color: got %v, want 0.2", got)
	}
	if got := writes.Load(); got != 2 {
		t.Fatalf("expected structure and content batches, got %d writes", got)
	}

	previousReads := reads.Load()
	if err := d.Apply(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if got := reads.Load(); got != previousReads+1 {
		t.Fatalf("next apply did not reload the layout exactly once: got %d reads, previously %d", got, previousReads)
	}
	if got := d.styles[styleBold].ForegroundColor.OpaqueColor.RgbColor.Red; got != 0.8 {
		t.Fatalf("next apply did not load the edited style: got %v, want 0.8", got)
	}
	if got := d.tableStyle.HeaderFirstCol.TextStyle.FontSize.Magnitude; got != 30 {
		t.Fatalf("next apply did not load the edited table font: got %v, want 30", got)
	}
	if got := d.shapes[styleBold].ShapeBackgroundFill.SolidFill.Color.RgbColor.Red; got != 0.8 {
		t.Fatalf("next apply did not load the edited shape: got %v, want 0.8", got)
	}
	if got := d.tableStyle.BorderStyle.OuterHorizontal.Weight.Magnitude; got != 3 {
		t.Fatalf("next apply did not load the edited border: got %v, want 3", got)
	}
	if got := writes.Load(); got != 2 {
		t.Fatalf("changes outside the comparison scope rewrote the table: %d writes", got)
	}

	backend.Layouts[1].PageElements = nil
	if err := d.Apply(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if len(d.styles) != 0 || len(d.shapes) != 0 {
		t.Fatal("removed template styles survived the next apply")
	}
	if d.tableStyle.HeaderFirstCol.TextStyle.FontSize != nil {
		t.Fatal("removed table template did not restore the default style")
	}
}

func TestApplyStyleSnapshotReleasedOnError(t *testing.T) {
	t.Parallel()

	backend := styleSnapshotPresentation()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("unchanged table must not be updated")
			http.Error(w, "unexpected update", http.StatusBadRequest)
			return
		}
		if err := json.NewEncoder(w).Encode(backend); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	srv, err := slides.NewService(context.Background(),
		option.WithEndpoint(server.URL+"/"), option.WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	d := &Deck{id: "presentation", srv: srv, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := d.Apply(context.Background(), Slides{{Layout: "Missing"}}); err == nil {
		t.Fatal("expected a layout validation error")
	}
	if d.stylesFrozen {
		t.Fatal("failed apply left the style snapshot frozen")
	}
	backend.Layouts[1].PageElements[0].Shape.Text.TextElements[0].TextRun.Style.ForegroundColor.OpaqueColor.RgbColor.Red = 0.8
	target := Slides{{
		Layout: "Body",
		Tables: []*Table{{Rows: []*TableRow{{Cells: []*TableCell{{
			IsHeader: true, Alignment: "START", Fragments: []*Fragment{{Value: "Header"}},
		}}}}}},
	}}
	if err := d.Apply(context.Background(), target); err != nil {
		t.Fatal(err)
	}
	if got := d.styles[styleBold].ForegroundColor.OpaqueColor.RgbColor.Red; got != 0.8 {
		t.Fatalf("next apply retained the failed snapshot: got %v, want 0.8", got)
	}
	if d.stylesFrozen {
		t.Fatal("successful apply left the style snapshot frozen")
	}
}

func styleSnapshotPresentation() *slides.Presentation {
	rgb := func(red float64) *slides.OpaqueColor {
		return &slides.OpaqueColor{RgbColor: &slides.RgbColor{Red: red}}
	}
	styleCell := func() *slides.TableCell {
		return &slides.TableCell{
			TableCellProperties: &slides.TableCellProperties{
				TableCellBackgroundFill: &slides.TableCellBackgroundFill{
					SolidFill: &slides.SolidFill{Color: rgb(0.2)},
				},
			},
			Text: &slides.TextContent{TextElements: []*slides.TextElement{{
				TextRun: &slides.TextRun{
					Content: "style\n",
					Style:   &slides.TextStyle{Bold: true, FontSize: &slides.Dimension{Magnitude: 14, Unit: "PT"}},
				},
			}}},
		}
	}
	border := func() *slides.TableBorderCell {
		return &slides.TableBorderCell{TableBorderProperties: &slides.TableBorderProperties{
			Weight: &slides.Dimension{Magnitude: 1, Unit: "PT"},
		}}
	}
	return &slides.Presentation{
		Layouts: []*slides.Page{
			{ObjectId: "layout", LayoutProperties: &slides.LayoutProperties{DisplayName: "Body"}},
			{
				ObjectId: "style-layout", LayoutProperties: &slides.LayoutProperties{DisplayName: layoutNameForStyle},
				PageElements: []*slides.PageElement{
					{Shape: &slides.Shape{
						Text: &slides.TextContent{TextElements: []*slides.TextElement{{
							TextRun: &slides.TextRun{
								Content: styleBold,
								Style: &slides.TextStyle{
									Italic: true, ForegroundColor: &slides.OptionalColor{OpaqueColor: rgb(0.2)},
								},
							},
						}}},
						ShapeProperties: &slides.ShapeProperties{
							ShapeBackgroundFill: &slides.ShapeBackgroundFill{
								SolidFill: &slides.SolidFill{Color: rgb(0.2)},
							},
						},
					}},
					{Table: &slides.Table{
						TableRows: []*slides.TableRow{
							{TableCells: []*slides.TableCell{styleCell(), styleCell()}},
							{TableCells: []*slides.TableCell{styleCell(), styleCell()}},
						},
						HorizontalBorderRows: []*slides.TableBorderRow{
							{TableBorderCells: []*slides.TableBorderCell{border(), border()}},
							{TableBorderCells: []*slides.TableBorderCell{border(), border()}},
							{TableBorderCells: []*slides.TableBorderCell{border(), border()}},
						},
					}},
				},
			},
		},
		Slides: []*slides.Page{{
			ObjectId: "slide",
			SlideProperties: &slides.SlideProperties{
				LayoutObjectId: "layout",
				NotesPage: &slides.Page{PageElements: []*slides.PageElement{{
					ObjectId: "notes", Shape: &slides.Shape{Placeholder: &slides.Placeholder{Type: "BODY"}},
				}}},
			},
			PageElements: []*slides.PageElement{{
				ObjectId: "table", Description: descriptionTableFromMarkdown,
				Table: &slides.Table{TableRows: []*slides.TableRow{{TableCells: []*slides.TableCell{{
					Text: &slides.TextContent{TextElements: []*slides.TextElement{
						{ParagraphMarker: &slides.ParagraphMarker{Style: &slides.ParagraphStyle{Alignment: "START"}}},
						{TextRun: &slides.TextRun{Content: "Header\n", Style: &slides.TextStyle{Bold: true}}},
					}},
				}}}}},
			}},
		}},
	}
}
