package deck

import (
	"bytes"
	"encoding/json"
)

type actionType int

const (
	actionTypeAppend actionType = iota // Append slide to the end
	actionTypeInsert                   // Insert slide at a specific index
	actionTypeUpdate                   // Update existing slide at a specific index
	actionTypeMove                     // Move existing slide to a new index
	actionTypeDelete                   // Delete slide at a specific index (not used in this diff)
)

func (at actionType) String() string {
	switch at {
	case actionTypeAppend:
		return "append"
	case actionTypeInsert:
		return "insert"
	case actionTypeUpdate:
		return "update"
	case actionTypeMove:
		return "move"
	case actionTypeDelete:
		return "delete"
	default:
		return "unknown"
	}
}

type action struct {
	actionType  actionType
	index       int
	moveToIndex int
	slide       *Slide
}

func diffSlides(before, after Slides) ([]*action, error) {
	return nil, nil // 実装は後で行います
}

// slideScore represents a slide with its similarity score
type slideScore struct {
	slide *Slide
	score int
	index int
}

// adjustSlideCount adjusts the count of before and after slides to be equal
// for Hungarian algorithm application
func adjustSlideCount(before, after Slides) (Slides, Slides, error) {
	if len(before) == len(after) {
		// No adjustment needed
		return before, after, nil
	}

	adjustedBefore := make(Slides, len(before))
	adjustedAfter := make(Slides, len(after))

	// Deep copy original slides
	for i, slide := range before {
		adjustedBefore[i] = deepCopySlide(slide)
	}
	for i, slide := range after {
		adjustedAfter[i] = deepCopySlide(slide)
	}

	if len(after) < len(before) {
		// after is shorter - add slides to after with .new = true
		return adjustShorterAfter(adjustedBefore, adjustedAfter)
	} else {
		// before is shorter - add slides to before with .delete = true
		return adjustShorterBefore(adjustedBefore, adjustedAfter)
	}
}

// adjustShorterAfter adds slides to after when it's shorter than before
func adjustShorterAfter(before, after Slides) (Slides, Slides, error) {
	needed := len(before) - len(after)

	// Calculate similarity scores for each before slide
	var scores []slideScore
	for i, beforeSlide := range before {
		totalScore := calculateTotalSimilarityScore(beforeSlide, after)
		scores = append(scores, slideScore{
			slide: beforeSlide,
			score: totalScore,
			index: i,
		})
	}

	// Sort by score (ascending - lowest similarity first)
	sortSlideScores(scores)

	// Add the slides with lowest similarity scores to after
	for i := 0; i < needed; i++ {
		slideToAdd := deepCopySlide(scores[i].slide)
		slideToAdd.new = true
		after = append(after, slideToAdd)
	}

	return before, after, nil
}

// adjustShorterBefore adds slides to before when it's shorter than after
func adjustShorterBefore(before, after Slides) (Slides, Slides, error) {
	needed := len(after) - len(before)

	// Calculate similarity scores for each after slide
	var scores []slideScore
	for i, afterSlide := range after {
		totalScore := calculateTotalSimilarityScore(afterSlide, before)
		scores = append(scores, slideScore{
			slide: afterSlide,
			score: totalScore,
			index: i,
		})
	}

	// Sort by score (ascending - lowest similarity first)
	sortSlideScores(scores)

	// Add the slides with lowest similarity scores to before
	for i := 0; i < needed; i++ {
		slideToAdd := deepCopySlide(scores[i].slide)
		slideToAdd.delete = true
		before = append(before, slideToAdd)
	}

	return before, after, nil
}

// calculateTotalSimilarityScore calculates the total similarity score
// of a slide against all slides in the target slice
func calculateTotalSimilarityScore(slide *Slide, targetSlides Slides) int {
	totalScore := 0
	for _, targetSlide := range targetSlides {
		totalScore += getSimilarity(slide, targetSlide)
	}
	return totalScore
}

// deepCopySlide creates a deep copy of a slide using JSON marshal/unmarshal
func deepCopySlide(slide *Slide) *Slide {
	if slide == nil {
		return nil
	}

	// Marshal to JSON
	data, err := json.Marshal(slide)
	if err != nil {
		// Fallback to original slide if marshal fails
		return slide
	}

	// Unmarshal to new slide
	copied := &Slide{}
	err = json.Unmarshal(data, copied)
	if err != nil {
		// Fallback to original slide if unmarshal fails
		return slide
	}

	// Copy unexported fields manually
	copied.new = slide.new
	copied.delete = slide.delete

	return copied
}

// sortSlideScores sorts slide scores in ascending order (lowest score first)
func sortSlideScores(scores []slideScore) {
	for i := 0; i < len(scores)-1; i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].score > scores[j].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
}

func getSimilarity(beforeSlide, afterSlide *Slide) int {
	if beforeSlide == nil || afterSlide == nil {
		return 0
	}

	if slidesEqual(beforeSlide, afterSlide) {
		return 500
	}

	score := 0

	// レイアウト一致の場合のみスコアを計算
	if beforeSlide.Layout == afterSlide.Layout && beforeSlide.Layout != "" {
		score += 10 // レイアウト基本点

		if len(beforeSlide.Titles) > 0 && len(afterSlide.Titles) > 0 && titlesEqual(beforeSlide.Titles, afterSlide.Titles) {
			score += 80
		}

		if len(beforeSlide.Subtitles) > 0 && len(afterSlide.Subtitles) > 0 && subtitlesEqual(beforeSlide.Subtitles, afterSlide.Subtitles) {
			score += 20
		}

		if len(beforeSlide.Bodies) > 0 && len(afterSlide.Bodies) > 0 && bodiesEqual(beforeSlide.Bodies, afterSlide.Bodies) {
			score += 160
		}
	}

	return score
}

// getSimilarityForMapping: マッピング用の類似度計算（位置ボーナス付き）
func getSimilarityForMapping(beforeSlide, afterSlide *Slide, beforeIndex, afterIndex int) int {
	// 基本類似度を取得
	baseScore := getSimilarity(beforeSlide, afterSlide)

	// 位置ボーナスを追加
	var positionBonus int
	switch {
	case beforeIndex == afterIndex:
		positionBonus = 4 // 完全な位置一致
	case beforeIndex < afterIndex:
		positionBonus = 2 // beforeがafterより前（自然な順序）
	default:
		positionBonus = 0 // beforeがafterより後
	}

	return baseScore + positionBonus
}

func slidesEqual(slide1, slide2 *Slide) bool {
	if slide1 == nil || slide2 == nil {
		return slide1 == slide2
	}

	// JSON比較による完全一致チェック
	slide1B, err1 := json.Marshal(slide1)
	if err1 != nil {
		return false
	}

	slide2B, err2 := json.Marshal(slide2)
	if err2 != nil {
		return false
	}

	return bytes.Equal(slide1B, slide2B)
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

	bodies1B, err1 := json.Marshal(bodies1)
	if err1 != nil {
		return false
	}

	bodies2B, err2 := json.Marshal(bodies2)
	if err2 != nil {
		return false
	}

	return bytes.Equal(bodies1B, bodies2B)
}
