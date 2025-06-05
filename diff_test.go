package deck

import (
	"strings"
	"testing"
)

func TestDiffSlides(t *testing.T) {
	tests := []struct {
		name     string
		before   Slides
		after    Slides
		expected []*action
	}{
		{
			name:     "empty slides",
			before:   Slides{},
			after:    Slides{},
			expected: []*action{},
		},
		{
			name:   "add new slide",
			before: Slides{},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"New Slide"},
				},
			},
			expected: []*action{
				{
					actionType:    actionTypeAppend,
					index:         0,
					originalIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide"},
					},
				},
			},
		},
		{
			name: "delete slide",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Old Slide"},
				},
			},
			after:    Slides{},
			expected: []*action{
				// No actions expected as deletion is handled separately
			},
		},
		{
			name: "move slide",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Slide A"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide B"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Slide B"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide A"},
				},
			},
			expected: []*action{
				{
					actionType:    actionTypeMove,
					index:         0,
					originalIndex: 1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Slide B"},
					},
				},
				// Second move is optimized away as it's automatically handled by the first move
			},
		},
		{
			name: "update slide content",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Original Title"},
				},
			},
			after: Slides{
				{
					Layout:    "TITLE",
					Titles:    []string{"Original Title"},
					Subtitles: []string{"New Subtitle"},
				},
			},
			expected: []*action{
				{
					actionType:    actionTypeUpdate,
					index:         0,
					originalIndex: 0,
					slide: &Slide{
						Layout:    "TITLE",
						Titles:    []string{"Original Title"},
						Subtitles: []string{"New Subtitle"},
					},
				},
			},
		},
		{
			name: "complex changes",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Slide 1"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 2"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 3"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Slide 2"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"New Slide"},
				},
				{
					Layout:    "TITLE",
					Titles:    []string{"Slide 1"},
					Subtitles: []string{"Updated"},
				},
			},
			expected: []*action{
				// Move actions first
				{
					actionType:    actionTypeMove,
					index:         0,
					originalIndex: 1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Slide 2"},
					},
				},
				// Update actions (New Slide replaces existing slide at lowest available index)
				{
					actionType:    actionTypeUpdate,
					index:         1,
					originalIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide"},
					},
				},
				// Update actions (Slide 1 with subtitles is detected as update)
				{
					actionType:    actionTypeUpdate,
					index:         2,
					originalIndex: 2,
					slide: &Slide{
						Layout:    "TITLE",
						Titles:    []string{"Slide 1"},
						Subtitles: []string{"Updated"},
					},
				},
			},
		},
		{
			name: "sequential execution index adjustment",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Slide 1"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 2"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 3"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 4"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 5"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"New Slide A"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 2"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"New Slide B"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Slide 4"},
				},
			},
			expected: []*action{
				// Update actions (replace existing slides, using lowest available indices)
				{
					actionType:    actionTypeUpdate,
					index:         0,
					originalIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide A"},
					},
				},
				{
					actionType:    actionTypeUpdate,
					index:         2,
					originalIndex: 2,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide B"},
					},
				},
			},
		},
		{
			name: "complex move and index adjustment",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"A"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"B"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"C"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"D"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"D"}, // moved from index 3 to 0
				},
				{
					Layout: "TITLE",
					Titles: []string{"B"}, // moved from index 1 to 1 (no change)
				},
				{
					Layout: "TITLE",
					Titles: []string{"A"}, // moved from index 0 to 2
				},
			},
			expected: []*action{
				// Move actions (no deletion handling)
				{
					actionType:    actionTypeMove,
					index:         0,
					originalIndex: 3, // D moved from original index 3
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"D"},
					},
				},
				{
					actionType:    actionTypeMove,
					index:         2,
					originalIndex: 0, // A moved
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"A"},
					},
				},
			},
		},
		{
			name: "update first page and adjust indices",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"First Page"}, // This will be updated
				},
				{
					Layout: "TITLE",
					Titles: []string{"Second Page"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Third Page"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"New First Page"}, // Updated page at index 0
				},
				{
					Layout: "TITLE",
					Titles: []string{"Second Page"}, // No change
				},
				{
					Layout: "TITLE",
					Titles: []string{"Third Page"}, // No change
				},
			},
			expected: []*action{
				{
					actionType:    actionTypeUpdate,
					index:         0,
					originalIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New First Page"},
					},
				},
			},
		},
		{
			name: "remove first page with moves and adds",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"First Page"}, // This will be replaced
				},
				{
					Layout: "TITLE",
					Titles: []string{"Second Page"}, // This will move to index 1
				},
				{
					Layout: "TITLE",
					Titles: []string{"Third Page"}, // This will move to index 0
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Third Page"}, // Moved from index 2 to 0
				},
				{
					Layout: "TITLE",
					Titles: []string{"Second Page"}, // Moved from index 1 to 1
				},
				{
					Layout: "TITLE",
					Titles: []string{"New Page"}, // Added at index 2
				},
			},
			expected: []*action{
				// Move Third Page
				{
					actionType:    actionTypeMove,
					index:         0,
					originalIndex: 2,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Third Page"},
					},
				},
				// Update existing page at index 2
				{
					actionType:    actionTypeUpdate,
					index:         2,
					originalIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Page"},
					},
				},
			},
		},
		{
			name: "multiple first page deletions and complex moves",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Delete Me 1"}, // index 0 - will be replaced
				},
				{
					Layout: "TITLE",
					Titles: []string{"Delete Me 2"}, // index 1 - will be replaced
				},
				{
					Layout: "TITLE",
					Titles: []string{"Keep Me A"}, // index 2 - will move to index 1
				},
				{
					Layout: "TITLE",
					Titles: []string{"Keep Me B"}, // index 3 - will move to index 0
				},
			},
			after: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Keep Me B"}, // moved from index 3 to 0
				},
				{
					Layout: "TITLE",
					Titles: []string{"Keep Me A"}, // moved from index 2 to 1
				},
				{
					Layout: "TITLE",
					Titles: []string{"New Page"}, // added at index 2
				},
			},
			expected: []*action{
				// Move actions
				{
					actionType:    actionTypeMove,
					index:         0,
					originalIndex: 3, // Keep Me B
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Keep Me B"},
					},
				},
				{
					actionType:    actionTypeMove,
					index:         1,
					originalIndex: 2, // Keep Me A
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Keep Me A"},
					},
				},
				// Update existing page at index 2
				{
					actionType:    actionTypeUpdate,
					index:         2,
					originalIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Page"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := diffSlides(tt.before, tt.after)
			if err != nil {
				t.Fatalf("diffSlides() error = %v", err)
			}

			if len(actions) != len(tt.expected) {
				if tt.name == "remove first page with moves and adds" {
					t.Logf("Actual actions for %s:", tt.name)
					for i, action := range actions {
						t.Logf("  [%d] Type: %s, Index: %d, OriginalIndex: %d, Slide: %v",
							i, action.actionType.String(), action.index, action.originalIndex,
							action.slide.Titles)
					}
					t.Logf("Expected actions:")
					for i, expected := range tt.expected {
						t.Logf("  [%d] Type: %s, Index: %d, OriginalIndex: %d, Slide: %v",
							i, expected.actionType.String(), expected.index, expected.originalIndex,
							expected.slide.Titles)
					}
				}
				t.Fatalf("diffSlides() returned %d actions, expected %d", len(actions), len(tt.expected))
			}

			// Create a map for easier comparison since order might vary
			actionMap := make(map[string]*action)
			expectedMap := make(map[string]*action)

			for _, action := range actions {
				key := createActionKey(*action)
				actionMap[key] = action
			}

			for _, expected := range tt.expected {
				key := createActionKey(*expected)
				expectedMap[key] = expected
			}

			for key, expected := range expectedMap {
				actual, exists := actionMap[key]
				if !exists {
					t.Errorf("Expected action not found: %+v", expected)
					continue
				}

				if !compareActions(*actual, *expected) {
					t.Errorf("Action mismatch:\nActual:   %+v\nExpected: %+v", actual, expected)
				}
			}

			for key, actual := range actionMap {
				if _, exists := expectedMap[key]; !exists {
					t.Errorf("Unexpected action found: %+v", actual)
				}
			}
		})
	}
}

func TestGenerateSlideKey(t *testing.T) {
	tests := []struct {
		name     string
		slide    *Slide
		expected string
	}{
		{
			name:     "nil slide",
			slide:    nil,
			expected: "",
		},
		{
			name: "simple slide",
			slide: &Slide{
				Layout: "TITLE",
				Titles: []string{"Test Title"},
			},
			expected: "TITLE|T:Test Title|N:",
		},
		{
			name: "complex slide",
			slide: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 1", "Title 2"},
				Subtitles: []string{"Subtitle"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Bullet: BulletDash,
								Fragments: []*Fragment{
									{Value: "Fragment 1"},
									{Value: "Fragment 2"},
								},
							},
						},
					},
				},
				SpeakerNote: "Speaker note",
			},
			expected: "TITLE_AND_BODY|T:Title 1|T:Title 2|S:Subtitle|B:-Fragment 1Fragment 2|N:Speaker note",
		},
		{
			name: "slide with fragment styles",
			slide: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Styled Content"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Bullet: BulletNone,
								Fragments: []*Fragment{
									{
										Value: "Bold text",
										Bold:  true,
									},
									{
										Value:  "Italic text",
										Italic: true,
									},
									{
										Value: "Link text",
										Link:  "https://example.com",
									},
									{
										Value: "Code text",
										Code:  true,
									},
									{
										Value:         "Line break",
										SoftLineBreak: true,
									},
								},
							},
						},
					},
				},
			},
			expected: "TITLE_AND_BODY|T:Styled Content|B:Bold text|BOLDItalic text|ITALICLink text|LINK:https://example.comCode text|CODELine break|N:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlideKey(tt.slide)
			if result != tt.expected {
				t.Errorf("generateSlideKey() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestSlidesHaveSimilarContent(t *testing.T) {
	tests := []struct {
		name     string
		slide1   *Slide
		slide2   *Slide
		expected bool
	}{
		{
			name:     "nil slides",
			slide1:   nil,
			slide2:   nil,
			expected: false,
		},
		{
			name:   "one nil slide",
			slide1: nil,
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Test"},
			},
			expected: false,
		},
		{
			name: "same titles",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Same Title"},
			},
			expected: true,
		},
		{
			name: "different titles",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title 2"},
			},
			expected: false,
		},
		{
			name: "same layouts, no titles",
			slide1: &Slide{
				Layout: "TITLE",
			},
			slide2: &Slide{
				Layout: "TITLE",
			},
			expected: true,
		},
		{
			name: "different layouts, no titles",
			slide1: &Slide{
				Layout: "TITLE",
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := slidesHaveSimilarContent(tt.slide1, tt.slide2)
			if result != tt.expected {
				t.Errorf("slidesHaveSimilarContent() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Helper functions for testing

func createActionKey(action action) string {
	var key strings.Builder
	key.WriteString(string(rune(action.actionType)))
	key.WriteString(":")
	if action.slide != nil && len(action.slide.Titles) > 0 {
		key.WriteString(action.slide.Titles[0])
	}
	return key.String()
}

func compareActions(actual, expected action) bool {
	if actual.actionType != expected.actionType {
		return false
	}
	if actual.index != expected.index {
		return false
	}
	if actual.originalIndex != expected.originalIndex {
		return false
	}

	// Compare slides
	if actual.slide == nil && expected.slide == nil {
		return true
	}
	if actual.slide == nil || expected.slide == nil {
		return false
	}

	return compareSlidesContent(actual.slide, expected.slide)
}

func compareSlidesContent(slide1, slide2 *Slide) bool {
	if slide1.Layout != slide2.Layout {
		return false
	}

	if len(slide1.Titles) != len(slide2.Titles) {
		return false
	}
	for i, title := range slide1.Titles {
		if title != slide2.Titles[i] {
			return false
		}
	}

	if len(slide1.Subtitles) != len(slide2.Subtitles) {
		return false
	}
	for i, subtitle := range slide1.Subtitles {
		if subtitle != slide2.Subtitles[i] {
			return false
		}
	}

	if slide1.SpeakerNote != slide2.SpeakerNote {
		return false
	}

	return true
}
