package deck

import (
	"bytes"
	"encoding/json"
	"slices"
)

func (s Slides) Compare(other Slides) bool {
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
	slices.SortFunc(images1, func(a *Image, b *Image) int {
		return int(a.Checksum()) - int(b.Checksum())
	})
	slices.SortFunc(images2, func(a *Image, b *Image) int {
		return int(a.Checksum()) - int(b.Checksum())
	})
	for i, img := range images1 {
		if !img.Compare(images2[i]) {
			return false
		}
	}
	return true
}

func blockQuotesEqual(bq1, bq2 []*BlockQuote) bool {
	if len(bq1) != len(bq2) {
		return false
	}
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
	merged1B, err := json.Marshal(mergeFragments(paragraph1.Fragments))
	if err != nil {
		return false
	}
	merged2B, err := json.Marshal(mergeFragments(paragraph2.Fragments))
	if err != nil {
		return false
	}
	return bytes.Equal(merged1B, merged2B)
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
		if in[i].SoftLineBreak {
			value += "\n"
		}
		if i > 0 {
			// Merge with previous fragment if possible
			if in[i-1].Bold == in[i].Bold &&
				in[i-1].Italic == in[i].Italic &&
				in[i-1].Link == in[i].Link &&
				in[i-1].Code == in[i].Code &&
				in[i-1].ClassName == in[i].ClassName {
				merged[len(merged)-1].Value += value
				continue
			}
		}
		merged = append(merged, &Fragment{
			Value:         in[i].Value,
			Bold:          in[i].Bold,
			Italic:        in[i].Italic,
			Link:          in[i].Link,
			Code:          in[i].Code,
			SoftLineBreak: false,
			ClassName:     in[i].ClassName,
		})
	}
	return merged
}
