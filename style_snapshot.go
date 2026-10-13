package deck

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/api/slides/v1"
)

type styleSnapshot struct {
	TextStyles map[string]*slides.TextStyle
	Shapes     map[string]*slides.ShapeProperties
	TableStyle *TableStyle
}

func (d *Deck) loadStyles() {
	d.styles = map[string]*slides.TextStyle{}
	d.shapes = map[string]*slides.ShapeProperties{}
	d.tableStyle = defaultTableStyle()
	for _, layout := range d.presentation.Layouts {
		if layout.LayoutProperties.DisplayName != layoutNameForStyle {
			continue
		}
		for _, element := range layout.PageElements {
			if element.Shape != nil && element.Shape.Text != nil {
				for _, text := range element.Shape.Text.TextElements {
					if text.TextRun == nil {
						continue
					}
					name := strings.Trim(text.TextRun.Content, " \n")
					if name != "" {
						d.styles[name] = text.TextRun.Style
						d.shapes[name] = element.Shape.ShapeProperties
					}
				}
			}
			if element.Table != nil {
				if style := extractTableStyleFromLayout(element.Table); style != nil {
					d.tableStyle = style
				}
			}
		}
	}
}

func (d *Deck) snapshotStyles() error {
	data, err := json.Marshal(styleSnapshot{
		TextStyles: d.styles,
		Shapes:     d.shapes,
		TableStyle: d.tableStyle,
	})
	if err != nil {
		return fmt.Errorf("failed to encode style snapshot: %w", err)
	}
	var snapshot styleSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("failed to decode style snapshot: %w", err)
	}
	d.styles = snapshot.TextStyles
	d.shapes = snapshot.Shapes
	d.tableStyle = snapshot.TableStyle
	return nil
}
