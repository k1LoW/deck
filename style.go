package deck

import (
	"slices"
	"sort"
	"strings"

	"google.golang.org/api/slides/v1"
)

const (
	styleCode   = "code"
	styleBold   = "bold"
	styleItalic = "italic"
	styleLink   = "link"
	// Define styles by referring to the default styles for most browsers described on the MDN.
	// ref. https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements
	styleStrong           = "strong" // <strong> tag
	styleEm               = "em"     // <em> tag
	styleS                = "s"      // <s> strikethrough tag
	styleU                = "u"      // <u> unarticulated annotation (underline) tag
	styleSup              = "sup"    // <sup> superscript tag
	styleSub              = "sub"    // <sub> subscript tag
	styleVar              = "var"    // <var> variable tag
	styleKbd              = "kbd"    // <kbd> keyboard input tag
	styleSamp             = "samp"   // <samp> sample output tag
	defaultCodeFontFamily = "Noto Sans Mono"
)

var (
	italicStyleFunc = func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Italic: true,
			},
			Fields: "italic",
		}
	}
	boldStyleFunc = func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Bold: true,
			},
			Fields: "bold",
		}
	}
	monospaceStyleFunc = func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				FontFamily: defaultCodeFontFamily,
			},
			Fields: "fontFamily",
		}
	}
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
	styleBold:   boldStyleFunc,
	styleItalic: italicStyleFunc,
	styleStrong: boldStyleFunc,
	styleEm:     italicStyleFunc,
	styleS: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Strikethrough: true,
			},
			Fields: "strikethrough",
		}
	},
	styleU: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				Underline: true,
			},
			Fields: "underline",
		}
	},
	styleSup: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				BaselineOffset: "SUPERSCRIPT",
			},
			Fields: "baselineOffset",
		}
	},
	styleSub: func() *slides.UpdateTextStyleRequest {
		return &slides.UpdateTextStyleRequest{
			Style: &slides.TextStyle{
				BaselineOffset: "SUBSCRIPT",
			},
			Fields: "baselineOffset",
		}
	},
	styleVar:  italicStyleFunc,
	styleKbd:  monospaceStyleFunc,
	styleSamp: monospaceStyleFunc,
}

func (d *Deck) getInlineStyleRequest(fragment *Fragment) *slides.UpdateTextStyleRequest {
	var reqs []*slides.UpdateTextStyleRequest

	if fragment.Code {
		reqs = append(reqs, d.getRequestForStyle(styleCode))
	}

	if fragment.Bold {
		reqs = append(reqs, d.getRequestForStyle(styleBold))
	}

	if fragment.Italic {
		reqs = append(reqs, d.getRequestForStyle(styleItalic))
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
		r := d.getRequestForStyle(fragment.StyleName)
		if r != nil {
			reqs = append(reqs, r)
		}
	}

	if len(reqs) == 0 {
		return nil
	}

	var (
		fields string
		style  *slides.TextStyle
	)
	for _, r := range reqs {
		// Merge elements with the latter taking priority.
		fields = mergeFields(fields, r.Fields)
		style = mergeStyles(style, r.Style, r.Fields)
	}

	return &slides.UpdateTextStyleRequest{
		Style:  style,
		Fields: fields,
	}
}

func (d *Deck) getRequestForStyle(styleName string) *slides.UpdateTextStyleRequest {
	if s, ok := d.styles[styleName]; ok {
		return buildCustomStyleRequest(s)
	}
	if f, ok := defaultStyles[styleName]; ok {
		return f()
	}
	return nil
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
			BaselineOffset:  s.BaselineOffset,
			Strikethrough:   s.Strikethrough,
		},
		Fields: "bold,italic,underline,foregroundColor,fontFamily,backgroundColor,baselineOffset,strikethrough",
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
	if slices.Contains(fields, "baselineOffset") {
		a.BaselineOffset = b.BaselineOffset
	}
	if slices.Contains(fields, "strikethrough") {
		a.Strikethrough = b.Strikethrough
	}
	return a
}
