package deck

import (
	"strings"
)

type actionType int

const (
	actionTypeAppend actionType = iota // Append slide to the end
	actionTypeInsert                   // Insert slide at a specific index
	actionTypeUpdate                   // Update existing slide at a specific index
	actionTypeMove                     // Move existing slide to a new index
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
	default:
		return "unknown"
	}
}

type action struct {
	actionType    actionType
	index         int
	originalIndex int
	slide         *Slide
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
					actionType:    actionTypeMove,
					index:         i,
					originalIndex: originalIndex,
					slide:         afterSlide,
				})
			}
			processedBefore[key] = true
			processedAfter[key] = true
		} else {
			// No exact match - look for similar content to update
			updated := false
			for beforeKey, beforeSlide := range beforeSlides {
				if processedBefore[beforeKey] {
					continue
				}

				// Simple heuristic: if slides have similar content, consider it an update
				if slidesHaveSimilarContent(beforeSlide, afterSlide) {
					originalIndex := beforeMap[beforeKey]
					actions = append(actions, &action{
						actionType:    actionTypeUpdate,
						index:         i,
						originalIndex: originalIndex,
						slide:         afterSlide,
					})
					processedBefore[beforeKey] = true
					processedAfter[key] = true
					updated = true
					break
				}
			}

			if !updated {
				// No similar content found - find the first unprocessed slide by index to update
				minOriginalIndex := -1
				var selectedKey string
				for beforeKey := range beforeSlides {
					if processedBefore[beforeKey] {
						continue
					}
					originalIndex := beforeMap[beforeKey]
					if minOriginalIndex == -1 || originalIndex < minOriginalIndex {
						minOriginalIndex = originalIndex
						selectedKey = beforeKey
					}
				}
				if minOriginalIndex != -1 {
					actions = append(actions, &action{
						actionType:    actionTypeUpdate,
						index:         i,
						originalIndex: minOriginalIndex,
						slide:         afterSlide,
					})
					processedBefore[selectedKey] = true
					processedAfter[key] = true
					updated = true
				}
			}
		}
	}

	// Second pass: handle additional slides beyond existing page count (add only when pages are insufficient)
	for i := minLength; i < len(after); i++ {
		afterSlide := after[i]
		key := generateSlideKey(afterSlide)

		// Only add new slides when we exceed the original page count
		actions = append(actions, &action{
			actionType:    actionTypeAppend,
			index:         i,
			originalIndex: -1, // No original index for new slides
			slide:         afterSlide,
		})
		processedAfter[key] = true
	}

	// Note: Removed slides are not handled here as deletion is handled separately
	// Only add, update, and move operations are processed for page adjustment from the beginning

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

	// Separate actions by type (excluding delete actions)
	var moveActions []*action
	var updateActions []*action
	var addActions []*action

	for _, action := range actions {
		switch action.actionType {
		case actionTypeMove:
			moveActions = append(moveActions, action)
		case actionTypeUpdate:
			updateActions = append(updateActions, action)
		case actionTypeAppend, actionTypeInsert:
			addActions = append(addActions, action)
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
		currentPos := currentPositions[move.originalIndex]
		targetPos := move.index

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
		currentPositions[move.originalIndex] = targetPos
	}

	return optimizedMoves
}

// generateSlideKey creates a unique key for a slide based on its content
func generateSlideKey(slide *Slide) string {
	if slide == nil {
		return ""
	}

	var key strings.Builder
	key.WriteString(slide.Layout)

	for _, title := range slide.Titles {
		key.WriteString("|T:")
		key.WriteString(title)
	}

	for _, subtitle := range slide.Subtitles {
		key.WriteString("|S:")
		key.WriteString(subtitle)
	}

	for _, body := range slide.Bodies {
		key.WriteString("|B:")
		for _, paragraph := range body.Paragraphs {
			key.WriteString(string(paragraph.Bullet))
			for _, fragment := range paragraph.Fragments {
				key.WriteString(fragment.Value)
				if fragment.Bold {
					key.WriteString("|BOLD")
				}
				if fragment.Italic {
					key.WriteString("|ITALIC")
				}
				if fragment.Link != "" {
					key.WriteString("|LINK:")
					key.WriteString(fragment.Link)
				}
				if fragment.Code {
					key.WriteString("|CODE")
				}
			}
		}
	}

	key.WriteString("|N:")
	key.WriteString(slide.SpeakerNote)
	return key.String()
}

// slidesHaveSimilarContent checks if two slides have similar content (for update detection)
func slidesHaveSimilarContent(slide1, slide2 *Slide) bool {
	if slide1 == nil || slide2 == nil {
		return false
	}

	// Check if titles match
	if len(slide1.Titles) > 0 && len(slide2.Titles) > 0 {
		return slide1.Titles[0] == slide2.Titles[0]
	}

	// Check if layouts match
	if slide1.Layout != "" && slide2.Layout != "" {
		return slide1.Layout == slide2.Layout
	}

	return false
}
