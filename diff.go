package deck

import (
	"bytes"
	"encoding/json"
	"fmt"
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

// mapSlides は before と after のスライドを1:1でマッピングする
// 前提: before と after の長さは同じ（adjustSlideCount で調整済み）
// 返り値: map[int]int - beforeのindexをキー、afterのindexを値とするマッピング
func mapSlides(before, after Slides) (map[int]int, error) {
	if len(before) != len(after) {
		return nil, fmt.Errorf("before and after slides must have the same length: before=%d, after=%d", len(before), len(after))
	}

	n := len(before)
	if n == 0 {
		return make(map[int]int), nil
	}

	// 類似度マトリックスを作成
	similarityMatrix := createSimilarityMatrix(before, after)

	// ハンガリアンアルゴリズムを実行（最大化問題として）
	assignment := hungarianAlgorithm(similarityMatrix)

	// 結果をmap[int]int形式に変換
	result := make(map[int]int)
	for beforeIdx, afterIdx := range assignment {
		result[beforeIdx] = afterIdx
	}

	return result, nil
}

// createSimilarityMatrix は類似度マトリックスを作成する
func createSimilarityMatrix(before, after Slides) [][]int {
	n := len(before)
	matrix := make([][]int, n)

	for i := 0; i < n; i++ {
		matrix[i] = make([]int, n)
		for j := 0; j < n; j++ {
			matrix[i][j] = getSimilarityForMapping(before[i], after[j], i, j)
		}
	}

	return matrix
}

// hungarianAlgorithm はハンガリアンアルゴリズムを実装（最大化問題用）
// 入力: 類似度マトリックス（値が大きいほど良いマッチ）
// 出力: assignment[i] = j は、beforeのi番目がafterのj番目にマッピングされることを意味
func hungarianAlgorithm(similarityMatrix [][]int) []int {
	n := len(similarityMatrix)
	if n == 0 {
		return []int{}
	}

	// 最大化問題を最小化問題に変換するため、最大値から各要素を引く
	maxValue := 0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if similarityMatrix[i][j] > maxValue {
				maxValue = similarityMatrix[i][j]
			}
		}
	}

	// コスト行列を作成（最大値 - 類似度）
	costMatrix := make([][]int, n)
	for i := 0; i < n; i++ {
		costMatrix[i] = make([]int, n)
		for j := 0; j < n; j++ {
			costMatrix[i][j] = maxValue - similarityMatrix[i][j]
		}
	}

	// ハンガリアンアルゴリズムの実装
	return solveAssignment(costMatrix)
}

// solveAssignment はハンガリアンアルゴリズムでコスト最小化問題を解く
func solveAssignment(costMatrix [][]int) []int {
	n := len(costMatrix)

	// 作業用のコピーを作成
	matrix := make([][]int, n)
	for i := 0; i < n; i++ {
		matrix[i] = make([]int, n)
		copy(matrix[i], costMatrix[i])
	}

	// Step 1: 各行から最小値を引く
	for i := 0; i < n; i++ {
		minVal := matrix[i][0]
		for j := 1; j < n; j++ {
			if matrix[i][j] < minVal {
				minVal = matrix[i][j]
			}
		}
		for j := 0; j < n; j++ {
			matrix[i][j] -= minVal
		}
	}

	// Step 2: 各列から最小値を引く
	for j := 0; j < n; j++ {
		minVal := matrix[0][j]
		for i := 1; i < n; i++ {
			if matrix[i][j] < minVal {
				minVal = matrix[i][j]
			}
		}
		for i := 0; i < n; i++ {
			matrix[i][j] -= minVal
		}
	}

	// Step 3: 最小数の線で全ての0をカバーできるかチェック
	for {
		assignment := findAssignment(matrix)
		if assignment != nil {
			return assignment
		}

		// 最小カバーを見つけて行列を更新
		if !updateMatrix(matrix) {
			break
		}
	}

	// フォールバック: 貪欲アルゴリズム
	return greedyAssignment(costMatrix)
}

// findAssignment は現在の行列で完全マッチングを見つける
func findAssignment(matrix [][]int) []int {
	n := len(matrix)
	assignment := make([]int, n)
	for i := 0; i < n; i++ {
		assignment[i] = -1
	}

	usedCols := make([]bool, n)

	// 各行で0の要素を1つだけ持つ行から開始
	for i := 0; i < n; i++ {
		zeroCount := 0
		zeroCol := -1
		for j := 0; j < n; j++ {
			if matrix[i][j] == 0 && !usedCols[j] {
				zeroCount++
				zeroCol = j
			}
		}
		if zeroCount == 1 {
			assignment[i] = zeroCol
			usedCols[zeroCol] = true
		}
	}

	// 残りの行を処理
	for i := 0; i < n; i++ {
		if assignment[i] == -1 {
			for j := 0; j < n; j++ {
				if matrix[i][j] == 0 && !usedCols[j] {
					assignment[i] = j
					usedCols[j] = true
					break
				}
			}
		}
	}

	// 全ての行が割り当てられているかチェック
	for i := 0; i < n; i++ {
		if assignment[i] == -1 {
			return nil
		}
	}

	return assignment
}

// updateMatrix は行列を更新して次のイテレーションに備える
func updateMatrix(matrix [][]int) bool {
	n := len(matrix)

	// 最小カバーを見つける（簡略化版）
	rowCovered := make([]bool, n)
	colCovered := make([]bool, n)

	// 0の要素がある行をマーク
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if matrix[i][j] == 0 {
				rowCovered[i] = true
				break
			}
		}
	}

	// カバーされていない要素の最小値を見つける
	minUncovered := -1
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if !rowCovered[i] && !colCovered[j] {
				if minUncovered == -1 || matrix[i][j] < minUncovered {
					minUncovered = matrix[i][j]
				}
			}
		}
	}

	if minUncovered == -1 {
		return false
	}

	// 行列を更新
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if rowCovered[i] && colCovered[j] {
				matrix[i][j] += minUncovered
			} else if !rowCovered[i] && !colCovered[j] {
				matrix[i][j] -= minUncovered
			}
		}
	}

	return true
}

// greedyAssignment は貪欲アルゴリズムで近似解を求める
func greedyAssignment(costMatrix [][]int) []int {
	n := len(costMatrix)
	assignment := make([]int, n)
	usedCols := make([]bool, n)

	for i := 0; i < n; i++ {
		bestCol := -1
		bestCost := -1

		for j := 0; j < n; j++ {
			if !usedCols[j] && (bestCol == -1 || costMatrix[i][j] < bestCost) {
				bestCol = j
				bestCost = costMatrix[i][j]
			}
		}

		assignment[i] = bestCol
		usedCols[bestCol] = true
	}

	return assignment
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
		// after is shorter - add slides to after with .delete = true
		return adjustShorterAfter(adjustedBefore, adjustedAfter)
	} else {
		// before is shorter - add slides to before with .new = true
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
		slideToAdd.delete = true
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
		slideToAdd.new = true
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

// markDeletedSlides applies delete marks from after slides to corresponding before slides
// based on the provided mapping. It modifies the before slides in-place.
func markDeletedSlides(before, after Slides, mapping map[int]int) {
	for beforeIdx, afterIdx := range mapping {
		if beforeIdx < len(before) && afterIdx < len(after) {
			if after[afterIdx].delete {
				before[beforeIdx].delete = true
			}
		}
	}
}
