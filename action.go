package deck

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"sort"

	"github.com/k1LoW/errors"
)

type actionType int

const (
	actionTypeAppend actionType = iota // Append slide to the end
	actionTypeUpdate                   // Update existing slide at a specific index
	actionTypeMove                     // Move existing slide to a new index
	actionTypeDelete                   // Delete slide at a specific index (not used in this diff)
)

func (at actionType) String() string { //nostyle:recvtype
	switch at {
	case actionTypeAppend:
		return "append"
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

func generateActions(before, after Slides) (_ []*action, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	// First, deep copy before and after slides
	beforeCopy, err := copySlides(before)
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy before slides: %w", err)
	}

	afterCopy, err := copySlides(after)
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy after slides: %w", err)
	}

	// Adjust slide count
	adjustedBefore, adjustedAfter, err := adjustSlideCount(beforeCopy, afterCopy)
	if err != nil {
		return nil, fmt.Errorf("failed to adjust slide count: %w", err)
	}

	// Map slides algorithm
	mapping, err := mapSlides(adjustedBefore, adjustedAfter)
	if err != nil {
		return nil, fmt.Errorf("failed to map slides: %w", err)
	}

	// Apply delete marks
	applyDeleteMarks(adjustedBefore, adjustedAfter, mapping)

	var actions []*action

	// Generate append actions first (before any updates that might reference new indices)
	appendActions := generateAppendActions(adjustedBefore)
	actions = append(actions, appendActions...)

	// Generate update actions (after append, so indices are stable)
	updateActions := generateUpdateActions(adjustedBefore, adjustedAfter, mapping)
	actions = append(actions, updateActions...)

	// Remove .delete slides from after
	cleanedAfter := removeDeleteMarked(adjustedAfter)

	// Generate delete actions
	deleteActions := generateDeleteActions(&adjustedBefore, &mapping)
	actions = append(actions, deleteActions...)

	// Generate move actions (executed with before after deletion and cleanedAfter)
	// At this point, deleted slides are also removed from before, so lengths match
	moveActions := generateMoveActions(&adjustedBefore, cleanedAfter, &mapping)
	actions = append(actions, moveActions...)

	return actions, nil
}

// mapSlides maps before and after slides 1:1
// Prerequisite: before and after have the same length (adjusted by adjustSlideCount)
// Returns: map[int]int - mapping with before index as key and after index as value.
func mapSlides(before, after Slides) (_ map[int]int, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	if len(before) != len(after) {
		return nil, fmt.Errorf("before and after slides must have the same length: before=%d, after=%d", len(before), len(after))
	}

	n := len(before)
	if n == 0 {
		return make(map[int]int), nil
	}

	// Create similarity matrix
	similarityMatrix := createSimilarityMatrix(before, after)

	// Execute Hungarian algorithm (as maximization problem)
	assignment := hungarianAlgorithm(similarityMatrix)

	// Convert result to map[int]int format
	result := make(map[int]int)
	for beforeIdx, afterIdx := range assignment {
		result[beforeIdx] = afterIdx
	}

	return result, nil
}

// createSimilarityMatrix creates a similarity matrix.
func createSimilarityMatrix(before, after Slides) [][]int {
	n := len(before)
	matrix := make([][]int, n)

	for i := range n {
		matrix[i] = make([]int, n)
		for j := range n {
			matrix[i][j] = getSimilarityForMapping(before[i], after[j], i, j)
		}
	}

	return matrix
}

// hungarianAlgorithm implements the Hungarian algorithm (for maximization problems)
// Input: similarity matrix (higher values indicate better matches)
// Output: assignment[i] = j means the i-th element of before is mapped to the j-th element of after.
func hungarianAlgorithm(similarityMatrix [][]int) []int {
	n := len(similarityMatrix)
	if n == 0 {
		return []int{}
	}

	// Convert maximization problem to minimization problem by subtracting each element from the maximum value
	maxValue := 0
	for i := range n {
		for j := range n {
			if similarityMatrix[i][j] > maxValue {
				maxValue = similarityMatrix[i][j]
			}
		}
	}

	// Create cost matrix (maximum value - similarity)
	costMatrix := make([][]int, n)
	for i := range n {
		costMatrix[i] = make([]int, n)
		for j := range n {
			costMatrix[i][j] = maxValue - similarityMatrix[i][j]
		}
	}

	// Implementation of Hungarian algorithm
	return solveAssignment(costMatrix)
}

// solveAssignment solves the cost minimization problem using the Hungarian algorithm.
func solveAssignment(costMatrix [][]int) []int {
	n := len(costMatrix)

	// Create working copy
	matrix := make([][]int, n)
	for i := range n {
		matrix[i] = make([]int, n)
		copy(matrix[i], costMatrix[i])
	}

	// Step 1: Subtract minimum value from each row
	for i := range n {
		minVal := matrix[i][0]
		for j := 1; j < n; j++ {
			if matrix[i][j] < minVal {
				minVal = matrix[i][j]
			}
		}
		for j := range n {
			matrix[i][j] -= minVal
		}
	}

	// Step 2: Subtract minimum value from each column
	for j := range n {
		minVal := matrix[0][j]
		for i := 1; i < n; i++ {
			if matrix[i][j] < minVal {
				minVal = matrix[i][j]
			}
		}
		for i := range n {
			matrix[i][j] -= minVal
		}
	}

	// Step 3: Check if all zeros can be covered with minimum number of lines
	for {
		assignment := findAssignment(matrix)
		if assignment != nil {
			return assignment
		}

		// Find minimum cover and update matrix
		if !updateMatrix(matrix) {
			break
		}
	}

	// Fallback: greedy algorithm
	return greedyAssignment(costMatrix)
}

// findAssignment finds a perfect matching in the current matrix.
func findAssignment(matrix [][]int) []int {
	n := len(matrix)
	assignment := make([]int, n)
	for i := range n {
		assignment[i] = -1
	}

	usedCols := make([]bool, n)

	// Start with rows that have only one zero element
	for i := range n {
		zeroCount := 0
		zeroCol := -1
		for j := range n {
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

	// Process remaining rows
	for i := range n {
		if assignment[i] == -1 {
			for j := range n {
				if matrix[i][j] == 0 && !usedCols[j] {
					assignment[i] = j
					usedCols[j] = true
					break
				}
			}
		}
	}

	// Check if all rows are assigned
	for i := range n {
		if assignment[i] == -1 {
			return nil
		}
	}

	return assignment
}

// updateMatrix updates the matrix for the next iteration.
func updateMatrix(matrix [][]int) bool {
	n := len(matrix)

	// Find minimum cover (simplified version)
	rowCovered := make([]bool, n)
	colCovered := make([]bool, n)

	// Mark rows that have zero elements
	for i := range n {
		for j := range n {
			if matrix[i][j] == 0 {
				rowCovered[i] = true
				break
			}
		}
	}

	// Find minimum value of uncovered elements
	minUncovered := -1
	for i := range n {
		for j := range n {
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

	// Update matrix
	for i := range n {
		for j := range n {
			if rowCovered[i] && colCovered[j] {
				matrix[i][j] += minUncovered
			} else if !rowCovered[i] && !colCovered[j] {
				matrix[i][j] -= minUncovered
			}
		}
	}

	return true
}

// greedyAssignment finds an approximate solution using greedy algorithm.
func greedyAssignment(costMatrix [][]int) []int {
	n := len(costMatrix)
	assignment := make([]int, n)
	usedCols := make([]bool, n)

	for i := range n {
		bestCol := -1
		bestCost := -1

		for j := range n {
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

// slideScore represents a slide with its similarity score.
type slideScore struct {
	slide *Slide
	score int
	index int
}

// adjustSlideCount adjusts the count of before and after slides to be equal
// for Hungarian algorithm application.
func adjustSlideCount(before, after Slides) (_ Slides, _ Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	if len(before) == len(after) {
		// No adjustment needed
		return before, after, nil
	}

	adjustedBefore := make(Slides, len(before))
	adjustedAfter := make(Slides, len(after))

	// Deep copy original slides
	for i, slide := range before {
		adjustedBefore[i] = copySlide(slide)
	}
	for i, slide := range after {
		adjustedAfter[i] = copySlide(slide)
	}

	if len(after) < len(before) {
		// after is shorter - add slides to after with .delete = true
		return adjustShorterAfter(adjustedBefore, adjustedAfter)
	} else {
		// before is shorter - add slides to before with .new = true
		return adjustShorterBefore(adjustedBefore, adjustedAfter)
	}
}

// adjustShorterAfter adds slides to after when it's shorter than before.
func adjustShorterAfter(before, after Slides) (_ Slides, _ Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

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
	slices.SortStableFunc(scores, func(a, b slideScore) int {
		return cmp.Compare(a.score, b.score)
	})

	// Add the slides with lowest similarity scores to after
	for i := range needed {
		slideToAdd := copySlide(scores[i].slide)
		slideToAdd.delete = true
		after = append(after, slideToAdd)
	}

	return before, after, nil
}

// adjustShorterBefore adds slides to before when it's shorter than after.
func adjustShorterBefore(before, after Slides) (_ Slides, _ Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

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
	slices.SortStableFunc(scores, func(a, b slideScore) int {
		return cmp.Compare(a.score, b.score)
	})

	// Add the slides with lowest similarity scores to before
	// But preserve the original order by adding them in index order
	indicesToAdd := make([]int, needed)
	for i := range needed {
		indicesToAdd[i] = scores[i].index
	}

	// Sort indices to maintain order
	sort.IntSlice(indicesToAdd).Sort()

	// Add slides in order
	for _, idx := range indicesToAdd {
		slideToAdd := copySlide(after[idx])
		slideToAdd.new = true
		before = append(before, slideToAdd)
	}

	return before, after, nil
}

// calculateTotalSimilarityScore calculates the total similarity score
// of a slide against all slides in the target slice.
func calculateTotalSimilarityScore(slide *Slide, targetSlides Slides) int {
	totalScore := 0
	for _, targetSlide := range targetSlides {
		totalScore += getSimilarity(slide, targetSlide)
	}
	return totalScore
}

// copySlide creates a deep copy of a slide using JSON marshal/unmarshal.
func copySlide(slide *Slide) *Slide {
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

	for _, image := range slide.Images {
		if image == nil {
			continue
		}
		for _, copiedImage := range copied.Images {
			if image.Compare(copiedImage) {
				copiedImage.fromMarkdown = image.fromMarkdown
				copiedImage.modTime = image.modTime
			}
		}
	}
	return copied
}

// copySlides creates a deep copy of slides using JSON marshal/unmarshal.
func copySlides(slides Slides) (_ Slides, err error) {
	defer func() {
		err = errors.WithStack(err)
	}()

	if slides == nil {
		return nil, nil
	}

	// Marshal to JSON
	data, err := json.Marshal(slides)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slides: %w", err)
	}

	// Unmarshal to new slides
	var copied Slides
	if err := json.Unmarshal(data, &copied); err != nil {
		return nil, fmt.Errorf("failed to unmarshal slides: %w", err)
	}

	for i, slide := range slides {
		for _, image := range slide.Images {
			for _, copiedImage := range copied[i].Images {
				if image.Compare(copiedImage) {
					copiedImage.fromMarkdown = image.fromMarkdown
					copiedImage.modTime = image.modTime
				}
			}
		}
	}

	return copied, nil
}

func getSimilarity(beforeSlide, afterSlide *Slide) int {
	if beforeSlide == nil || afterSlide == nil {
		return 0
	}

	if slidesEqual(beforeSlide, afterSlide) {
		return 500
	}

	score := 0

	// Calculate score only when layouts match
	if beforeSlide.Layout == afterSlide.Layout && beforeSlide.Layout != "" {
		score += 50 // Increased layout base score from 10 to 50

		if len(beforeSlide.Titles) > 0 && len(afterSlide.Titles) > 0 && titlesEqual(beforeSlide.Titles, afterSlide.Titles) {
			score += 80
		}

		if len(beforeSlide.Subtitles) > 0 && len(afterSlide.Subtitles) > 0 && subtitlesEqual(beforeSlide.Subtitles, afterSlide.Subtitles) {
			score += 20
		}

		if len(beforeSlide.Bodies) > 0 && len(afterSlide.Bodies) > 0 && bodiesEqual(beforeSlide.Bodies, afterSlide.Bodies) {
			score += 160
		}

		if len(beforeSlide.Images) > 0 && len(afterSlide.Images) > 0 && imagesEqual(beforeSlide.Images, afterSlide.Images) {
			score += 40
		}

		if len(beforeSlide.BlockQuotes) > 0 && len(afterSlide.BlockQuotes) > 0 && blockQuotesEqual(beforeSlide.BlockQuotes, afterSlide.BlockQuotes) {
			score += 30
		}
	}

	return score
}

// getSimilarityForMapping: similarity calculation for mapping (with position bonus).
func getSimilarityForMapping(beforeSlide, afterSlide *Slide, beforeIndex, afterIndex int) int {
	// Get base similarity
	baseScore := getSimilarity(beforeSlide, afterSlide)

	// Add position bonus - prioritize earlier positions for same layout
	var positionBonus int
	if beforeSlide.Layout == afterSlide.Layout && beforeSlide.Layout != "" {
		// For same layout, prefer earlier positions in after
		switch {
		case beforeIndex == afterIndex:
			positionBonus = 8 // Perfect position match
		case afterIndex < beforeIndex:
			positionBonus = 6 // Prefer earlier positions in after
		case beforeIndex < afterIndex:
			positionBonus = 4 // Natural order
		default:
			positionBonus = 2 // Default bonus
		}
	} else {
		// For different layouts, use original logic
		switch {
		case beforeIndex == afterIndex:
			positionBonus = 4 // Perfect position match
		case beforeIndex < afterIndex:
			positionBonus = 2 // before is ahead of after (natural order)
		default:
			positionBonus = 0 // before is behind after
		}
	}

	return baseScore + positionBonus
}

// generateUpdateActions generates update actions.
func generateUpdateActions(before, after Slides, mapping map[int]int) []*action {
	var actions []*action

	for beforeIdx, afterIdx := range mapping {
		if beforeIdx >= len(before) || afterIdx >= len(after) {
			continue // Ignore out of range
		}

		beforeSlide := before[beforeIdx]
		afterSlide := after[afterIdx]

		// Don't update slides marked with .delete
		if beforeSlide.delete {
			continue
		}

		// Generate update action if similarity is less than 500
		// Include .new slides if they are mapped to specific after slides
		score := getSimilarity(beforeSlide, afterSlide)
		if score < 500 {
			actions = append(actions, &action{
				actionType: actionTypeUpdate,
				index:      beforeIdx,
				slide:      afterSlide,
			})
		}
	}

	return actions
}

// generateAppendActions generates append actions.
func generateAppendActions(before Slides) []*action {
	var actions []*action

	for i, slide := range before {
		if slide.new {
			// Generate append action for slides marked with .new
			// append adds to the end, so index is not needed
			actions = append(actions, &action{
				actionType: actionTypeAppend,
				index:      i, // Original index in before (for reference)
				slide:      slide,
			})
		}
	}

	return actions
}

// removeDeleteMarked removes slides marked with .delete.
func removeDeleteMarked(after Slides) Slides {
	var cleaned Slides

	for _, slide := range after {
		if !slide.delete {
			cleaned = append(cleaned, slide)
		}
	}

	return cleaned
}

// generateDeleteActions generates delete actions and actually removes slides from before
// Also updates mapping simultaneously.
func generateDeleteActions(before *Slides, mapping *map[int]int) []*action {
	var actions []*action

	// Delete from back to maintain index consistency
	for i := len(*before) - 1; i >= 0; i-- {
		slide := (*before)[i]
		if slide.delete {
			// Generate delete action (add in deletion order)
			actions = append(actions, &action{
				actionType: actionTypeDelete,
				index:      i,
				slide:      slide,
			})

			// Actually delete the slide
			*before = slices.Delete(*before, i, i+1)

			// Update mapping: adjust mappings after the deleted index
			updateMappingAfterDeletion(mapping, i)
		}
	}

	return actions
}

// updateMappingAfterDeletion updates mappings after the deleted index.
func updateMappingAfterDeletion(mapping *map[int]int, deletedIndex int) {
	newMapping := make(map[int]int)

	for beforeIdx, afterIdx := range *mapping {
		if beforeIdx < deletedIndex {
			// Mappings before the deleted index remain unchanged
			newMapping[beforeIdx] = afterIdx
		} else if beforeIdx > deletedIndex {
			// Mappings after the deleted index are decremented by 1
			newMapping[beforeIdx-1] = afterIdx
		}
		// If beforeIdx == deletedIndex, remove (exclude from mapping)
	}

	*mapping = newMapping
}

// generateMoveActions generates move actions and actually moves slides in before
// Uses efficient algorithm to generate minimum move actions.
func generateMoveActions(before *Slides, after Slides, mapping *map[int]int) []*action {
	var actions []*action

	// Don't process if current before and after have different lengths
	if len(*before) != len(after) {
		return actions
	}

	if len(*before) == 0 {
		return actions
	}

	// Assign unique IDs to slides for tracking
	type slideWithID struct {
		slide      *Slide
		originalID int
	}

	// Create working slide list (assign unique IDs)
	workingSlides := make([]slideWithID, len(*before))
	for i, slide := range *before {
		workingSlides[i] = slideWithID{
			slide:      copySlide(slide),
			originalID: i,
		}
	}

	// Create reverse mapping (afterIndex -> beforeIndex)
	reverseMapping := make(map[int]int)
	for beforeIdx, afterIdx := range *mapping {
		reverseMapping[afterIdx] = beforeIdx
	}

	// Create target arrangement (record originalID that should be at each position)
	targetOrder := make([]int, len(after))
	for targetPos := range len(after) {
		if expectedBeforeIdx, exists := reverseMapping[targetPos]; exists {
			targetOrder[targetPos] = expectedBeforeIdx
		} else {
			targetOrder[targetPos] = -1 // Invalid position
		}
	}

	// Compare current arrangement with target arrangement to identify necessary moves
	for targetPos := range len(targetOrder) {
		expectedOriginalID := targetOrder[targetPos]
		if expectedOriginalID == -1 {
			continue
		}

		// Check originalID of slide currently at this position
		currentOriginalID := workingSlides[targetPos].originalID

		// Skip if correct slide is already placed
		if currentOriginalID == expectedOriginalID {
			continue
		}

		// Find where the correct slide is located
		correctPos := -1
		for i, slideWithID := range workingSlides {
			if slideWithID.originalID == expectedOriginalID {
				correctPos = i
				break
			}
		}

		if correctPos != -1 && correctPos != targetPos {
			// Generate move action
			actions = append(actions, &action{
				actionType:  actionTypeMove,
				index:       correctPos,
				moveToIndex: targetPos,
				slide:       copySlide(workingSlides[correctPos].slide),
			})

			// Execute move in working slides
			slideToMove := workingSlides[correctPos]

			// Remove slide
			workingSlides = slices.Delete(workingSlides, correctPos, correctPos+1)

			// Adjust insertion position
			insertIndex := targetPos
			if targetPos > correctPos {
				insertIndex = targetPos - 1
			}

			// Insert at specified position
			if insertIndex >= len(workingSlides) {
				workingSlides = append(workingSlides, slideToMove)
			} else {
				workingSlides = append(workingSlides[:insertIndex], append([]slideWithID{slideToMove}, workingSlides[insertIndex:]...)...)
			}
		}
	}

	// Reflect final state to before
	finalSlides := make(Slides, len(workingSlides))
	for i, slideWithID := range workingSlides {
		finalSlides[i] = slideWithID.slide
	}
	*before = finalSlides

	return actions
}

// applyDeleteMarks applies delete marks from after slides to corresponding before slides
// based on the provided mapping. It modifies the before slides in-place.
func applyDeleteMarks(before, after Slides, mapping map[int]int) {
	for beforeIdx, afterIdx := range mapping {
		if beforeIdx < len(before) && afterIdx < len(after) {
			if after[afterIdx].delete {
				before[beforeIdx].delete = true
			}
		}
	}
}
