package deck

import (
	"slices"
	"sort"
	"strings"

	"google.golang.org/api/slides/v1"
)

const (
	styleCode             = "code"
	styleBold             = "bold"
	styleItalic           = "italic"
	styleLink             = "link"
	styleBlockQuote       = "blockquote"
	defaultCodeFontFamily = "Noto Sans Mono"
)

var defaultStyles = map[string]func() *slides.UpdateTextStyleRequest{
	styleCode: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				ForegroundColor: &slides.OptionalColor{
					OpaqueColor: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{
							Red:   0.0,
							Green: 0.0,
							Blue:  0.0,
						},
					},
				},
				FontFamily: defaultCodeFontFamily,
				BackgroundColor: &slides.OptionalColor{
					OpaqueColor: &slides.OpaqueColor{
						RgbColor: &slides.RgbColor{
							Red:   0.95,
							Green: 0.95,
							Blue:  0.95,
						},
					},
				},
			},
			Fields: "foregroundColor,fontFamily,backgroundColor",
		}
	},
	styleBold: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Bold: true,
			},
			Fields: "bold",
		}
	},
	styleItalic: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Italic: true,
			},
			Fields: "italic",
		}
	},
}

func (d *Deck) getInlineStyleRequests(fragment *Fragment) (reqs []*slides.UpdateTextStyleRequest) {
	if fragment.Code {
		s, ok := d.styles[styleCode]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, defaultStyles[styleCode]())
		}
	}

	if fragment.Bold {
		s, ok := d.styles[styleBold]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, defaultStyles[styleBold]())
		}
	}

	if fragment.Italic {
		s, ok := d.styles[styleItalic]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		} else {
			reqs = append(reqs, defaultStyles[styleItalic]())
		}
	}

	if fragment.Link != "" {
		s, ok := d.styles[styleLink]
		if ok {
			req := buildCustomStyleRequest(s)
			req.Fields = "link,bold,italic,underline,foregroundColor,fontFamily,backgroundColor"
			req.Style.Link = &slides.Link{
				Url: fragment.Link,
			}
			reqs = append(reqs, req)
		} else {
			reqs = append(reqs, &slides.UpdateTextStyleRequest{
				Style: &slides.TextStyle{
					Link: &slides.Link{
						Url: fragment.Link,
					},
				},
				Fields: "link",
			})
		}
	}

	if fragment.StyleName != "" {
		s, ok := d.styles[fragment.StyleName]
		if ok {
			reqs = append(reqs, buildCustomStyleRequest(s))
		}
	}

	return reqs
}

func buildCustomStyleRequest(s *slides.TextStyle) *slides.UpdateTextStyleRequest {
	return &slides.UpdateTextStyleRequest{
		Style: &slides.TextStyle{
			Bold:            s.Bold,
			Italic:          s.Italic,
			Underline:       s.Underline,
			ForegroundColor: s.ForegroundColor,
			FontFamily:      s.FontFamily,
			BackgroundColor: s.BackgroundColor,
		},
		Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor",
	}
}

func mergeFields(a, b string) string {
	fields := strings.Split(a, ",")
	fields = append(fields, strings.Split(b, ",")...)
	sort.Strings(fields)
	fields = slices.Compact(fields)
	return strings.Join(fields, ",")
}

func mergeStyles(a, b *slides.TextStyle, fStr string) *slides.TextStyle {
	if a == nil {
		return b
	}
	fields := strings.Split(fStr, ",")
	if slices.Contains(fields, "link") {
		a.Link = b.Link
	}
	if slices.Contains(fields, "bold") {
		a.Bold = b.Bold
	}
	if slices.Contains(fields, "italic") {
		a.Italic = b.Italic
	}
	if slices.Contains(fields, "underline") {
		a.Underline = b.Underline
	}
	if slices.Contains(fields, "foregroundColor") {
		a.ForegroundColor = b.ForegroundColor
	}
	if slices.Contains(fields, "fontFamily") {
		a.FontFamily = b.FontFamily
	}
	if slices.Contains(fields, "backgroundColor") {
		a.BackgroundColor = b.BackgroundColor
	}
	return a
}
