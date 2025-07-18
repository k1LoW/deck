package deck

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
)

func (s Slides) Compare(other Slides) bool { //nostyle:recvtype
	if len(s) != len(other) {
		return false
	}
	for i := range s {
		if !slidesEqual(s[i], other[i]) {
			return false
		}
	}
	return true
}

func (s *Slide) Compare(other *Slide) bool {
	if s == nil || other == nil {
		return s == other
	}
	if s.Layout != other.Layout {
		return false
	}
	if !titlesEqual(s.Titles, other.Titles) {
		return false
	}
	if !subtitlesEqual(s.Subtitles, other.Subtitles) {
		return false
	}
	if !bodiesEqual(s.Bodies, other.Bodies) {
		return false
	}
	if !imagesEqual(s.Images, other.Images) {
		return false
	}
	if !blockQuotesEqual(s.BlockQuotes, other.BlockQuotes) {
		return false
	}
	if s.SpeakerNote != other.SpeakerNote {
		return false
	}
	return true
}

func slidesEqual(slide1, slide2 *Slide) bool {
	return slide1.Compare(slide2)
}

func titlesEqual(titles1, titles2 []string) bool {
	if len(titles1) != len(titles2) {
		return false
	}
	for i := range titles1 {
		if titles1[i] != titles2[i] {
			return false
		}
	}
	return true
}

func subtitlesEqual(subtitles1, subtitles2 []string) bool {
	if len(subtitles1) != len(subtitles2) {
		return false
	}
	for i := range subtitles1 {
		if subtitles1[i] != subtitles2[i] {
			return false
		}
	}
	return true
}

func bodiesEqual(bodies1, bodies2 []*Body) bool {
	if len(bodies1) != len(bodies2) {
		return false
	}
	for i := range bodies1 {
		if bodies1[i] == nil || bodies2[i] == nil {
			if bodies1[i] != bodies2[i] {
				return false
			}
		}
		if !paragraphsEqual(bodies1[i].Paragraphs, bodies2[i].Paragraphs) {
			return false
		}
	}
	return true
}

func imagesEqual(images1, images2 []*Image) bool {
	if len(images1) != len(images2) {
		return false
	}
	sorted1 := make([]*Image, len(images1))
	copy(sorted1, images1)
	sorted2 := make([]*Image, len(images2))
	copy(sorted2, images2)

	slices.SortFunc(sorted1, func(a *Image, b *Image) int {
		return int(a.Checksum()) - int(b.Checksum())
	})
	slices.SortFunc(sorted2, func(a *Image, b *Image) int {
		return int(a.Checksum()) - int(b.Checksum())
	})
	for i, img := range sorted1 {
		if !img.Compare(sorted2[i]) {
			return false
		}
	}
	return true
}

func blockQuotesEqual(bq1, bq2 []*BlockQuote) bool {
	if len(bq1) != len(bq2) {
		return false
	}
	slices.SortFunc(bq1, func(a *BlockQuote, b *BlockQuote) int {
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
	})
	slices.SortFunc(bq2, func(a *BlockQuote, b *BlockQuote) int {
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
	})

	for i := range bq1 {
		if bq1[i] == nil || bq2[i] == nil {
			if bq1[i] != bq2[i] {
				return false
			}
		}
		if bq1[i].Nesting != bq2[i].Nesting {
			return false
		}
		if !paragraphsEqual(bq1[i].Paragraphs, bq2[i].Paragraphs) {
			return false
		}
	}
	return true
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
	if len(merged1) != len(merged2) {
		return false
	}
	for i := range merged1 {
		if strings.TrimRight(merged1[i].Value, "\n") != strings.TrimRight(merged2[i].Value, "\n") {
			return false
		}
		if merged1[i].Bold != merged2[i].Bold ||
			merged1[i].Italic != merged2[i].Italic ||
			merged1[i].Link != merged2[i].Link ||
			merged1[i].Code != merged2[i].Code {
			return false
		}
	}
	return true
}

func paragraphsEqual(paragraphs1, paragraphs2 []*Paragraph) bool {
	if len(paragraphs1) != len(paragraphs2) {
		return false
	}
	for i := range paragraphs1 {
		if !paragraphEqual(paragraphs1[i], paragraphs2[i]) {
			return false
		}
	}
	return true
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
