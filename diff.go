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
