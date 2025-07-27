package deck

import (
	"bytes"
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
		imagesEqual(s.Images, other.Images) &&
		blockQuotesEqual(s.BlockQuotes, other.BlockQuotes) &&
		s.SpeakerNote == other.SpeakerNote
}

func bodiesEqual(bodies1, bodies2 []*Body) bool {
	return slices.EqualFunc(bodies1, bodies2, func(a, b *Body) bool {
		if a == nil || b == nil {
			return a == b
		}
		return slices.EqualFunc(a.Paragraphs, b.Paragraphs, paragraphEqual)
	})
}

func imagesEqual(images1, images2 []*Image) bool {
	if len(images1) != len(images2) {
		return false
	}
	sorted1 := make([]*Image, len(images1))
	copy(sorted1, images1)
	sorted2 := make([]*Image, len(images2))
	copy(sorted2, images2)

	f := func(a *Image, b *Image) int {
		return int(a.Checksum()) - int(b.Checksum())
	}
	slices.SortFunc(sorted1, f)
	slices.SortFunc(sorted2, f)

	return slices.EqualFunc(sorted1, sorted2, func(a, b *Image) bool {
		return a.Equivalent(b)
	})
}

func blockQuotesEqual(bq1, bq2 []*BlockQuote) bool {
	if len(bq1) != len(bq2) {
		return false
	}
	f := func(a *BlockQuote, b *BlockQuote) int {
		if a.Nesting != b.Nesting {
			return a.Nesting - b.Nesting
		}
		if len(a.Paragraphs) != len(b.Paragraphs) {
			return len(a.Paragraphs) - len(b.Paragraphs)
		}
		jsonA, err := json.Marshal(a.Paragraphs)
		if err != nil {
			return -1 // Error case, treat as unequal
		}
		jsonB, err := json.Marshal(b.Paragraphs)
		if err != nil {
			return 1 // Error case, treat as unequal
		}
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
	if paragraph1.Bullet != paragraph2.Bullet {
		return false
	}
	if paragraph1.Nesting != paragraph2.Nesting {
		return false
	}
	merged1 := mergeFragments(paragraph1.Fragments)
	merged2 := mergeFragments(paragraph2.Fragments)

	return slices.EqualFunc(merged1, merged2, func(a, b *Fragment) bool {
		return strings.TrimRight(a.Value, "\n") == strings.TrimRight(b.Value, "\n") &&
			a.Bold == b.Bold &&
			a.Italic == b.Italic &&
			a.Link == b.Link &&
			a.Code == b.Code
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
			if in[i-1].Bold == in[i].Bold &&
				in[i-1].Italic == in[i].Italic &&
				in[i-1].Link == in[i].Link &&
				in[i-1].Code == in[i].Code &&
				in[i-1].StyleName == in[i].StyleName {
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
