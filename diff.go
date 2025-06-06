package deck

import (
	"encoding/json"
	"time"
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
	var actions []*action

	// Create maps for efficient lookup with separate index tracking
	beforeMap := make(map[string]int)       // key -> original index
	afterMap := make(map[string]int)        // key -> new index
	beforeSlides := make(map[string]*Slide) // key -> slide
	afterSlides := make(map[string]*Slide)  // key -> slide

	// Generate unique keys for slides based on their content
	for i, slide := range before {
		key := generateSlideKey(slide)
		beforeMap[key] = i
		beforeSlides[key] = slide
	}

	for i, slide := range after {
		key := generateSlideKey(slide)
		afterMap[key] = i
		afterSlides[key] = slide
	}

	// Track processed slides
	processedBefore := make(map[string]bool)
	processedAfter := make(map[string]bool)
	// Track positions that have been updated (to avoid deleting slides at those positions)
	updatedPositions := make(map[int]bool)

	// Process slides up to the minimum of before and after lengths
	// Use move and update for existing page positions
	minLength := len(before)
	if len(after) < minLength {
		minLength = len(after)
	}

	// First pass: handle slides within existing page range
	for i := 0; i < minLength; i++ {
		afterSlide := after[i]
		key := generateSlideKey(afterSlide)

		if originalIndex, exists := beforeMap[key]; exists {
			// Exact match found - check if moved
			if originalIndex != i {
				actions = append(actions, &action{
					actionType:  actionTypeMove,
					index:       originalIndex,
					moveToIndex: i,
					slide:       afterSlide,
				})
			}
			processedBefore[key] = true
			processedAfter[key] = true
		} else {
			// No exact match - look for similar content to update with priority

			// Find slides with best similarity match based on priority
			var bestMatch struct {
				key           string
				slide         *Slide
				originalIndex int
				priority      int // 1=layout+title+subtitle, 2=layout+title, 3=layout+subtitle, 4=title only, 5=layout only, 6=subtitle only, 7=no match
			}
			bestMatch.priority = 8 // Initialize with lowest priority

			// Create a sorted list of before slides by index to ensure deterministic behavior
			type beforeSlideInfo struct {
				key           string
				slide         *Slide
				originalIndex int
			}
			var sortedBeforeSlides []beforeSlideInfo
			for beforeKey, beforeSlide := range beforeSlides {
				if !processedBefore[beforeKey] {
					sortedBeforeSlides = append(sortedBeforeSlides, beforeSlideInfo{
						key:           beforeKey,
						slide:         beforeSlide,
						originalIndex: beforeMap[beforeKey],
					})
				}
			}

			// Sort by original index to ensure deterministic behavior
			for i := 0; i < len(sortedBeforeSlides); i++ {
				for j := i + 1; j < len(sortedBeforeSlides); j++ {
					if sortedBeforeSlides[i].originalIndex > sortedBeforeSlides[j].originalIndex {
						sortedBeforeSlides[i], sortedBeforeSlides[j] = sortedBeforeSlides[j], sortedBeforeSlides[i]
					}
				}
			}

			for _, beforeInfo := range sortedBeforeSlides {
				priority := getSimilarityPriority(beforeInfo.slide, afterSlide)

				// Prefer slides at the same index when priority is equal, but only for layout matches
				if priority < bestMatch.priority ||
					(priority == bestMatch.priority && priority <= 2 && beforeInfo.originalIndex == i && bestMatch.originalIndex != i) {
					bestMatch.key = beforeInfo.key
					bestMatch.slide = beforeInfo.slide
					bestMatch.originalIndex = beforeInfo.originalIndex
					bestMatch.priority = priority
				}
			}

			if bestMatch.priority <= 5 { // Only match for layout/title/subtitle matches, not subtitle-only or no match
				// Special case: Use move for layout and title match when position changes
				// to avoid layout change issues in applyPage
				if bestMatch.priority <= 2 && bestMatch.originalIndex != i {
					actions = append(actions, &action{
						actionType:  actionTypeMove,
						index:       bestMatch.originalIndex,
						moveToIndex: i,
						slide:       bestMatch.slide,
					})
					// Then update the content if needed
					if generateSlideKey(bestMatch.slide) != generateSlideKey(afterSlide) {
						actions = append(actions, &action{
							actionType:  actionTypeUpdate,
							index:       i,
							moveToIndex: -1,
							slide:       afterSlide,
						})
					}
				} else {
					// Use update for other cases
					actions = append(actions, &action{
						actionType:  actionTypeUpdate,
						index:       i,
						moveToIndex: -1,
						slide:       afterSlide,
					})
				}
				processedBefore[bestMatch.key] = true
				processedAfter[key] = true
				updatedPositions[i] = true
			} else {
				// No suitable match found - update the slide at this position
				actions = append(actions, &action{
					actionType:  actionTypeUpdate,
					index:       i,
					moveToIndex: -1,
					slide:       afterSlide,
				})
				processedAfter[key] = true
				updatedPositions[i] = true
			}
		}
	}

	// Second pass: handle additional slides beyond existing page count (add only when pages are insufficient)
	for i := minLength; i < len(after); i++ {
		afterSlide := after[i]
		key := generateSlideKey(afterSlide)

		// Only add new slides when we exceed the original page count
		actions = append(actions, &action{
			actionType:  actionTypeAppend,
			index:       i,
			moveToIndex: -1,
			slide:       afterSlide,
		})
		processedAfter[key] = true
	}

	// Create a list of slides to delete with their original indices
	var slidesToDelete []struct {
		index int
		slide *Slide
	}

	// Create a sorted list of unprocessed before slides to ensure deterministic behavior
	type deleteSlideInfo struct {
		key           string
		slide         *Slide
		originalIndex int
	}
	var sortedDeleteSlides []deleteSlideInfo
	for beforeKey, beforeSlide := range beforeSlides {
		if !processedBefore[beforeKey] {
			originalIndex := beforeMap[beforeKey]
			// Skip slides at positions that have been updated
			if !updatedPositions[originalIndex] {
				sortedDeleteSlides = append(sortedDeleteSlides, deleteSlideInfo{
					key:           beforeKey,
					slide:         beforeSlide,
					originalIndex: originalIndex,
				})
			}
		}
	}

	// Sort by original index to ensure deterministic behavior
	for i := 0; i < len(sortedDeleteSlides); i++ {
		for j := i + 1; j < len(sortedDeleteSlides); j++ {
			if sortedDeleteSlides[i].originalIndex > sortedDeleteSlides[j].originalIndex {
				sortedDeleteSlides[i], sortedDeleteSlides[j] = sortedDeleteSlides[j], sortedDeleteSlides[i]
			}
		}
	}

	for _, deleteInfo := range sortedDeleteSlides {
		slidesToDelete = append(slidesToDelete, struct {
			index int
			slide *Slide
		}{deleteInfo.originalIndex, deleteInfo.slide})
	}

	// Sort slides to delete by index in descending order (highest index first)
	for i := 0; i < len(slidesToDelete); i++ {
		for j := i + 1; j < len(slidesToDelete); j++ {
			if slidesToDelete[i].index < slidesToDelete[j].index {
				slidesToDelete[i], slidesToDelete[j] = slidesToDelete[j], slidesToDelete[i]
			}
		}
	}

	// Add delete actions in the correct order
	for _, slideToDelete := range slidesToDelete {
		actions = append(actions, &action{
			actionType:  actionTypeDelete,
			index:       slideToDelete.index,
			moveToIndex: -1,
			slide:       slideToDelete.slide,
		})
	}

	// Sort and adjust actions for sequential execution
	return adjustActionsForSequentialExecution(actions, len(before)), nil
}

// adjustActionsForSequentialExecution sorts actions and adjusts indices for sequential execution
// Actions are ordered to process page adjustments from the beginning:
// 1. Move actions (to reposition existing slides, optimized to avoid redundant moves)
// 2. Update actions (to modify existing slides in their new positions)
// 3. Add actions (to insert new slides from lowest index to highest)
// Note: Delete actions are not processed here as they are handled separately
func adjustActionsForSequentialExecution(actions []*action, originalLength int) []*action {
	if len(actions) == 0 {
		return actions
	}

	// Separate actions by type
	var moveActions []*action
	var updateActions []*action
	var addActions []*action
	var deleteActions []*action

	for _, action := range actions {
		switch action.actionType {
		case actionTypeMove:
			moveActions = append(moveActions, action)
		case actionTypeUpdate:
			updateActions = append(updateActions, action)
		case actionTypeAppend, actionTypeInsert:
			addActions = append(addActions, action)
		case actionTypeDelete:
			deleteActions = append(deleteActions, action)
		}
	}

	var result []*action

	// 1. Process move actions with optimization
	// Sort move actions by target index to process from beginning to end
	for i := 0; i < len(moveActions); i++ {
		for j := i + 1; j < len(moveActions); j++ {
			if moveActions[i].index > moveActions[j].index {
				moveActions[i], moveActions[j] = moveActions[j], moveActions[i]
			}
		}
	}

	// Optimize move actions: simulate the moves and only include necessary ones
	optimizedMoves := optimizeMoveActions(moveActions, originalLength)
	result = append(result, optimizedMoves...)

	// 2. Process update actions
	// Sort update actions by target index
	for i := 0; i < len(updateActions); i++ {
		for j := i + 1; j < len(updateActions); j++ {
			if updateActions[i].index > updateActions[j].index {
				updateActions[i], updateActions[j] = updateActions[j], updateActions[i]
			}
		}
	}
	result = append(result, updateActions...)

	// 3. Process add actions from lowest index to highest
	// This ensures proper insertion order from the beginning
	for i := 0; i < len(addActions); i++ {
		for j := i + 1; j < len(addActions); j++ {
			if addActions[i].index > addActions[j].index {
				addActions[i], addActions[j] = addActions[j], addActions[i]
			}
		}
	}
	result = append(result, addActions...)

	// 4. Process delete actions from highest index to lowest
	// This ensures proper deletion order (delete from end to beginning)
	for i := 0; i < len(deleteActions); i++ {
		for j := i + 1; j < len(deleteActions); j++ {
			if deleteActions[i].index < deleteActions[j].index {
				deleteActions[i], deleteActions[j] = deleteActions[j], deleteActions[i]
			}
		}
	}
	result = append(result, deleteActions...)

	return result
}

// optimizeMoveActions optimizes move actions by simulating sequential execution
// and removing redundant moves that would be automatically handled by previous moves
func optimizeMoveActions(moveActions []*action, originalLength int) []*action {
	if len(moveActions) == 0 {
		return moveActions
	}

	// Create a simulation of the current slide positions
	// Map from original index to current index
	currentPositions := make(map[int]int)
	for i := 0; i < originalLength; i++ {
		currentPositions[i] = i
	}

	var optimizedMoves []*action

	// Process moves in order and simulate their effects
	for _, move := range moveActions {
		currentPos := currentPositions[move.index]
		targetPos := move.moveToIndex

		// If the slide is already in the correct position, skip this move
		if currentPos == targetPos {
			continue
		}

		// Add this move to the optimized list
		optimizedMoves = append(optimizedMoves, move)

		// Simulate the move: update all positions
		// When moving from currentPos to targetPos, all slides between them shift
		if currentPos < targetPos {
			// Moving forward: slides between currentPos+1 and targetPos shift left
			for origIdx, pos := range currentPositions {
				if pos > currentPos && pos <= targetPos {
					currentPositions[origIdx] = pos - 1
				}
			}
		} else {
			// Moving backward: slides between targetPos and currentPos-1 shift right
			for origIdx, pos := range currentPositions {
				if pos >= targetPos && pos < currentPos {
					currentPositions[origIdx] = pos + 1
				}
			}
		}
		// Update the moved slide's position
		currentPositions[move.index] = targetPos
	}

	return optimizedMoves
}

// generateSlideKey creates a unique key for a slide based on its content
func generateSlideKey(slide *Slide) string {
	b, err := json.Marshal(slide)
	if err != nil {
		return time.Now().String() // Fallback to current time if JSON marshalling fails
	}
	return string(b)
}

// slidesHaveSimilarContent checks if two slides have similar content (for update detection)
// Priority order:
// 1. Exact layout and title match (highest priority for reuse)
// 2. Title match only
// 3. Layout match only
func slidesHaveSimilarContent(slide1, slide2 *Slide) bool {
	if slide1 == nil || slide2 == nil {
		return false
	}

	// Check if both layout and titles match (highest priority)
	if slide1.Layout != "" && slide2.Layout != "" && slide1.Layout == slide2.Layout {
		if len(slide1.Titles) > 0 && len(slide2.Titles) > 0 {
			return slide1.Titles[0] == slide2.Titles[0]
		}
		// If layouts match but no titles, still consider it similar
		return true
	}

	// Check if titles match (medium priority)
	if len(slide1.Titles) > 0 && len(slide2.Titles) > 0 {
		return slide1.Titles[0] == slide2.Titles[0]
	}

	return false
}

// getSimilarityPriority returns the priority for slide similarity matching
// Lower numbers indicate higher priority for reuse
// 1: Exact layout, title, and subtitle match (highest priority)
// 2: Exact layout and title match
// 3: Exact layout and subtitle match
// 4: Title match only
// 5: Layout match only
// 6: Subtitle match only
// 7: No specific match (lowest priority)
func getSimilarityPriority(beforeSlide, afterSlide *Slide) int {
	if beforeSlide == nil || afterSlide == nil {
		return 6
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
