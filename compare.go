package deck

import (
	"bytes"
	"cmp"
	"encoding/json"
	"slices"
	"strings"
)

func (s Slides) Equal(other Slides) bool { //nostyle:recvtype
	return slices.EqualFunc(s, other, func(a, b *Slide) bool {
		return a.Equal(b)
	})
}

func (s *Slide) Equal(other *Slide) bool {
	if s == nil || other == nil {
		return s == other
	}
	return s.Layout == other.Layout &&
		slices.Equal(s.Titles, other.Titles) &&
		slices.Equal(s.Subtitles, other.Subtitles) &&
		bodiesEqual(s.Bodies, other.Bodies) &&
		imagesEquivalent(s.Images, other.Images) &&
		blockQuotesEqual(s.BlockQuotes, other.BlockQuotes) &&
		tablesEqual(s.Tables, other.Tables) &&
		s.SpeakerNote == other.SpeakerNote
}

func bodiesEqual(bodies1, bodies2 []*Body) bool {
	return slices.EqualFunc(bodies1, bodies2, func(a, b *Body) bool {
		return slices.EqualFunc(a.Paragraphs, b.Paragraphs, paragraphEqual)
	})
}

func imagesEquivalent(images1, images2 []*Image) bool {
	sorted1 := make([]*Image, len(images1))
	copy(sorted1, images1)
	sorted2 := make([]*Image, len(images2))
	copy(sorted2, images2)

	f := func(a *Image, b *Image) int {
		c := cmp.Compare(a.link, b.link)
		if c != 0 {
			return c
		}
		return int(a.Checksum()) - int(b.Checksum())
	}
	slices.SortFunc(sorted1, f)
	slices.SortFunc(sorted2, f)

	return slices.EqualFunc(sorted1, sorted2, func(a, b *Image) bool {
		return a.Equivalent(b)
	})
}

func blockQuotesEqual(bq1, bq2 []*BlockQuote) bool {
	f := func(a *BlockQuote, b *BlockQuote) int {
		if a.Nesting != b.Nesting {
			return a.Nesting - b.Nesting
		}
		jsonA, _ := json.Marshal(a.Paragraphs)
		jsonB, _ := json.Marshal(b.Paragraphs)
		return bytes.Compare(jsonA, jsonB)
	}
	slices.SortFunc(bq1, f)
	slices.SortFunc(bq2, f)

	return slices.EqualFunc(bq1, bq2, func(a, b *BlockQuote) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Nesting == b.Nesting &&
			slices.EqualFunc(a.Paragraphs, b.Paragraphs, paragraphEqual)
	})
}

func paragraphEqual(paragraph1, paragraph2 *Paragraph) bool {
	if paragraph1 == nil || paragraph2 == nil {
		return paragraph1 == paragraph2
	}
	if paragraph1.Bullet != paragraph2.Bullet || paragraph1.Nesting != paragraph2.Nesting {
		return false
	}
	merged1 := mergeFragments(paragraph1.Fragments)
	merged2 := mergeFragments(paragraph2.Fragments)

	return slices.EqualFunc(merged1, merged2, func(a, b *Fragment) bool {
		return strings.TrimRight(a.Value, "\n") == strings.TrimRight(b.Value, "\n") &&
			a.StylesEqual(b)
	})
}

func mergeFragments(in []*Fragment) []*Fragment {
	var merged []*Fragment
	if len(in) == 0 {
		return merged
	}
	for i := range len(in) {
		value := in[i].Value
		if i > 0 {
			// Merge with previous fragment if possible
			if in[i-1].StylesEqual(in[i]) {
				merged[len(merged)-1].Value += value
				continue
			}
		}
		merged = append(merged, &Fragment{
			Value:     in[i].Value,
			Bold:      in[i].Bold,
			Italic:    in[i].Italic,
			Link:      in[i].Link,
			Code:      in[i].Code,
			StyleName: in[i].StyleName,
		})
	}
	return merged
}

func tablesEqual(tables1, tables2 []*Table) bool {
	return slices.EqualFunc(tables1, tables2, func(a, b *Table) bool {
		if a == nil || b == nil {
			return a == b
		}
		return slices.EqualFunc(a.Rows, b.Rows, tableRowEqual)
	})
}

func tableRowEqual(row1, row2 *TableRow) bool {
	if row1 == nil || row2 == nil {
		return row1 == row2
	}
	return slices.EqualFunc(row1.Cells, row2.Cells, tableCellEqual)
}

func tableCellEqual(cell1, cell2 *TableCell) bool {
	if cell1 == nil || cell2 == nil {
		return cell1 == cell2
	}
	if cell1.Alignment != cell2.Alignment || cell1.IsHeader != cell2.IsHeader {
		return false
	}
	return slices.EqualFunc(cell1.Fragments, cell2.Fragments, func(a, b *Fragment) bool {
		return strings.TrimRight(a.Value, "\n") == strings.TrimRight(b.Value, "\n") &&
			a.StylesEqual(b)
	})
}
