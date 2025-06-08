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

// positionTracker tracks the current position of each slide after moves.
type positionTracker struct {
	currentPos   map[int]int // originalIndex -> currentPosition
	posContent   map[int]int // currentPosition -> originalIndex
	originalSize int
}

// newPositionTracker creates a new position tracker.
func newPositionTracker(size int) *positionTracker {
	tracker := &positionTracker{
		currentPos:   make(map[int]int),
		posContent:   make(map[int]int),
		originalSize: size,
	}

	// Initialize with identity mapping
	for i := 0; i < size; i++ {
		tracker.currentPos[i] = i
		tracker.posContent[i] = i
	}

	return tracker
}

// getCurrentPosition returns the current position of a slide by its original index.
func (pt *positionTracker) getCurrentPosition(originalIndex int) int {
	if pos, exists := pt.currentPos[originalIndex]; exists {
		return pos
	}
	return -1 // Not found
}

// getSlideAtPosition returns the original index of the slide at the given position.
func (pt *positionTracker) getSlideAtPosition(position int) int {
	if origIdx, exists := pt.posContent[position]; exists {
		return origIdx
	}
	return -1 // Not found
}

// moveSlide updates positions after a move operation.
func (pt *positionTracker) moveSlide(originalIndex, fromPos, toPos int) {
	// Remove from old position
	delete(pt.posContent, fromPos)

	// Shift slides between fromPos and toPos
	if fromPos < toPos {
		// Moving forward: slides between fromPos+1 and toPos shift left
		for pos := fromPos + 1; pos <= toPos; pos++ {
			if origIdx, exists := pt.posContent[pos]; exists {
				pt.posContent[pos-1] = origIdx
				pt.currentPos[origIdx] = pos - 1
			}
		}
	} else {
		// Moving backward: slides between toPos and fromPos-1 shift right
		for pos := fromPos - 1; pos >= toPos; pos-- {
			if origIdx, exists := pt.posContent[pos]; exists {
				pt.posContent[pos+1] = origIdx
				pt.currentPos[origIdx] = pos + 1
			}
		}
	}

	// Place moved slide at target position
	pt.posContent[toPos] = originalIndex
	pt.currentPos[originalIndex] = toPos
}

// removeSlide removes a slide from tracking (for delete operations).
func (pt *positionTracker) removeSlide(originalIndex int) {
	if pos, exists := pt.currentPos[originalIndex]; exists {
		delete(pt.currentPos, originalIndex)
		delete(pt.posContent, pos)
	}
}

func diffSlides(before, after Slides) ([]*action, error) {
	var actions []*action

	if len(before) == 0 && len(after) == 0 {
		return actions, nil
	}

	// Initialize position tracker
	tracker := newPositionTracker(len(before))

	// Track which slides have been processed
	processedBefore := make(map[int]bool)

	// Process each target position in order
	minLength := len(before)
	if len(after) < minLength {
		minLength = len(after)
	}

	// First pass: handle slides within existing page range
	for targetPos := 0; targetPos < minLength; targetPos++ {
		targetSlide := after[targetPos]

		// Find the best matching slide in before
		bestMatch := findBestMatchingSlide(targetSlide, before, processedBefore)

		if bestMatch.originalIndex == -1 {
			// No suitable match found - this will be an update operation
			// Check if the slide at this position has any similarity with the target
			slideAtPos := tracker.getSlideAtPosition(targetPos)
			if slideAtPos != -1 {
				slideAtPosition := before[slideAtPos]
				similarity := getSimilarityPriority(slideAtPosition, targetSlide)

				// Mark as processed if there's meaningful similarity
				// Priority 4 or better (title match or better), or subtitle match
				if similarity <= 4 || similarity == 6 {
					processedBefore[slideAtPos] = true
				} else if slideAtPos == targetPos || targetPos < minLength {
					// Mark as processed if:
					// 1. The slide is at its original position (no moves involved), OR
					// 2. We're updating within the overlapping range of before and after
					processedBefore[slideAtPos] = true
				}
				// Layout-only matches (priority 5) or no match (priority 7)
				// should not prevent deletion of the original slide unless it's a direct replacement
			}

			actions = append(actions, &action{
				actionType:  actionTypeUpdate,
				index:       targetPos,
				moveToIndex: -1,
				slide:       targetSlide,
			})
			continue
		}

		// Get current position of the best matching slide
		currentPos := tracker.getCurrentPosition(bestMatch.originalIndex)

		// If slide is not at target position, generate move action
		if currentPos != targetPos {
			actions = append(actions, &action{
				actionType:  actionTypeMove,
				index:       currentPos,
				moveToIndex: targetPos,
				slide:       bestMatch.slide,
			})

			// Update tracker after move
			tracker.moveSlide(bestMatch.originalIndex, currentPos, targetPos)
		}

		// Mark as processed
		processedBefore[bestMatch.originalIndex] = true

		// If content is different, add update action
		if !slidesEqual(bestMatch.slide, targetSlide) {
			actions = append(actions, &action{
				actionType:  actionTypeUpdate,
				index:       targetPos,
				moveToIndex: -1,
				slide:       targetSlide,
			})
		}
	}

	// Second pass: handle additional slides beyond existing page count
	for i := minLength; i < len(after); i++ {
		actions = append(actions, &action{
			actionType:  actionTypeAppend,
			index:       i,
			moveToIndex: -1,
			slide:       after[i],
		})
	}

	// Third pass: delete unprocessed slides
	var deleteActions []*action
	for originalIndex := 0; originalIndex < len(before); originalIndex++ {
		if !processedBefore[originalIndex] {
			currentPos := tracker.getCurrentPosition(originalIndex)
			if currentPos != -1 {
				deleteActions = append(deleteActions, &action{
					actionType:  actionTypeDelete,
					index:       currentPos,
					moveToIndex: -1,
					slide:       before[originalIndex],
				})
			}
		}
	}

	// Sort delete actions by position in descending order (delete from end to beginning)
	for i := 0; i < len(deleteActions); i++ {
		for j := i + 1; j < len(deleteActions); j++ {
			if deleteActions[i].index < deleteActions[j].index {
				deleteActions[i], deleteActions[j] = deleteActions[j], deleteActions[i]
			}
		}
	}

	actions = append(actions, deleteActions...)

	return actions, nil
}

// matchResult represents the result of slide matching.
type matchResult struct {
	originalIndex int
	slide         *Slide
	priority      int
}

// findBestMatchingSlide finds the best matching slide for the target.
func findBestMatchingSlide(targetSlide *Slide, before Slides, processedBefore map[int]bool) matchResult {
	bestMatch := matchResult{originalIndex: -1, priority: 8}

	for i, beforeSlide := range before {
		if processedBefore[i] {
			continue
		}

		priority := getSimilarityPriority(beforeSlide, targetSlide)

		// Accept matches with priority 5 or better (layout/title/subtitle matches)
		if priority <= 5 && priority < bestMatch.priority {
			bestMatch.originalIndex = i
			bestMatch.slide = beforeSlide
			bestMatch.priority = priority
		}
	}

	return bestMatch
}

// slidesEqual checks if two slides have identical content.
func slidesEqual(slide1, slide2 *Slide) bool {
	if slide1 == nil || slide2 == nil {
		return slide1 == slide2
	}

	// Use JSON marshaling for complete comparison
	// This ensures all fields including Bodies are compared
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

// getSimilarityPriority returns the priority for slide similarity matching
// Lower numbers indicate higher priority for reuse
// 0: Perfect match - identical content (highest priority)
// 1: Exact layout, title, and subtitle match
// 2: Exact layout and title match
// 3: Exact layout and subtitle match
// 4: Title match only
// 5: Layout match only
// 6: Subtitle match only
// 7: No specific match (lowest priority)
func getSimilarityPriority(beforeSlide, afterSlide *Slide) int {
	if beforeSlide == nil || afterSlide == nil {
		return 7
	}

	beforeB, err := json.Marshal(beforeSlide)
	if err != nil {
		return 7
	}
	afterB, err := json.Marshal(afterSlide)
	if err != nil {
		return 7
	}

	layoutMatch := beforeSlide.Layout != "" && afterSlide.Layout != "" && beforeSlide.Layout == afterSlide.Layout

	// Check all titles for match
	titleMatch := true
	if len(beforeSlide.Titles) != len(afterSlide.Titles) {
		titleMatch = false
	} else {
		for i := range beforeSlide.Titles {
			if beforeSlide.Titles[i] != afterSlide.Titles[i] {
				titleMatch = false
				break
			}
		}
	}

	// Check all subtitles for match (only if both slides have subtitles)
	subtitleMatch := false
	if len(beforeSlide.Subtitles) > 0 && len(afterSlide.Subtitles) > 0 {
		if len(beforeSlide.Subtitles) == len(afterSlide.Subtitles) {
			subtitleMatch = true
			for i := range beforeSlide.Subtitles {
				if beforeSlide.Subtitles[i] != afterSlide.Subtitles[i] {
					subtitleMatch = false
					break
				}
			}
		}
	}

	// Determine priority based on match combinations
	switch {
	case bytes.Equal(beforeB, afterB):
		return 0 // Perfect match: same content and position
	case layoutMatch && titleMatch && subtitleMatch:
		return 1 // Highest priority: layout, title, and subtitle all match (with actual subtitles)
	case layoutMatch && titleMatch:
		return 2 // High priority: both layout and title match
	case layoutMatch && subtitleMatch:
		return 3 // High priority: both layout and subtitle match
	case titleMatch:
		return 4 // Medium priority: title match only
	case layoutMatch:
		return 5 // Lower priority: layout match only
	case subtitleMatch:
		return 6 // Lower priority: subtitle match only
	default:
		return 7 // Lowest priority: no specific match
	}
}
