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
					actionType:  actionTypeAppend,
					index:       0,
					moveToIndex: -1,
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
			after: Slides{},
			expected: []*action{
				{
					actionType:  actionTypeDelete,
					index:       0,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Old Slide"},
					},
				},
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
					actionType:  actionTypeMove,
					index:       0,
					moveToIndex: 1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Slide A"},
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
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
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
					actionType:  actionTypeMove,
					index:       1,
					moveToIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Slide 2"},
					},
				},
				// Update actions (New Slide replaces existing slide at lowest available index)
				{
					actionType:  actionTypeUpdate,
					index:       1,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide"},
					},
				},
				// Update actions (Slide 1 with subtitles is detected as update)
				{
					actionType:  actionTypeUpdate,
					index:       2,
					moveToIndex: -1,
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
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide A"},
					},
				},
				{
					actionType:  actionTypeUpdate,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Slide B"},
					},
				},
				// Delete actions for unused slides
				{
					actionType:  actionTypeDelete,
					index:       4,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Slide 5"},
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
				// Move actions
				{
					actionType:  actionTypeMove,
					index:       3,
					moveToIndex: 0, // D moved from original index 3
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"D"},
					},
				},
				{
					actionType:  actionTypeMove,
					index:       0,
					moveToIndex: 2, // A moved
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"A"},
					},
				},
				// Delete action for unused slide
				{
					actionType:  actionTypeDelete,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"C"},
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
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
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
					actionType:  actionTypeMove,
					index:       2,
					moveToIndex: 0,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Third Page"},
					},
				},
				// Update existing page at index 2
				{
					actionType:  actionTypeUpdate,
					index:       2,
					moveToIndex: -1,
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
					actionType:  actionTypeMove,
					index:       3,
					moveToIndex: 0, // Keep Me B
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Keep Me B"},
					},
				},
				{
					actionType:  actionTypeMove,
					index:       2,
					moveToIndex: 1, // Keep Me A
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Keep Me A"},
					},
				},
				// Update existing page at index 2
				{
					actionType:  actionTypeUpdate,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"New Page"},
					},
				},
				// Delete action for unused slide
				{
					actionType:  actionTypeDelete,
					index:       1,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Delete Me 2"},
					},
				},
			},
		},
		{
			name: "reuse slide with same layout and title after move",
			before: Slides{
				{
					Layout: "TITLE",
					Titles: []string{"Same Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Bullet: BulletNone,
									Fragments: []*Fragment{
										{Value: "Original content"},
									},
								},
							},
						},
					},
				},
				{
					Layout: "TITLE_AND_BODY",
					Titles: []string{"Different Title"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Another Title"},
				},
			},
			after: Slides{
				{
					Layout: "TITLE_AND_BODY",
					Titles: []string{"Different Title"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Same Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Bullet: BulletNone,
									Fragments: []*Fragment{
										{Value: "Updated content"},
									},
								},
							},
						},
					},
				},
			},
			expected: []*action{
				// Move "Same Title" to index 1
				{
					actionType:  actionTypeMove,
					index:       0,
					moveToIndex: 1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Same Title"},
						Bodies: []*Body{
							{
								Paragraphs: []*Paragraph{
									{
										Bullet: BulletNone,
										Fragments: []*Fragment{
											{Value: "Original content"},
										},
									},
								},
							},
						},
					},
				},
				// Update "Same Title" slide (reuse existing slide with same layout and title)
				{
					actionType:  actionTypeUpdate,
					index:       1,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Same Title"},
						Bodies: []*Body{
							{
								Paragraphs: []*Paragraph{
									{
										Bullet: BulletNone,
										Fragments: []*Fragment{
											{Value: "Updated content"},
										},
									},
								},
							},
						},
					},
				},
				// Delete action for unused slide
				{
					actionType:  actionTypeDelete,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Another Title"},
					},
				},
			},
		},
		{
			name: "prioritize exact layout and title match for reuse",
			before: Slides{
				{
					Layout:    "TITLE",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"Old subtitle"},
				},
				{
					Layout: "TITLE_AND_BODY",
					Titles: []string{"Target Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Bullet: BulletNone,
									Fragments: []*Fragment{
										{Value: "Body content"},
									},
								},
							},
						},
					},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Other Title"},
				},
			},
			after: Slides{
				{
					Layout:    "TITLE",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"New subtitle"},
				},
				{
					Layout: "TITLE_AND_BODY",
					Titles: []string{"Target Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Bullet: BulletNone,
									Fragments: []*Fragment{
										{Value: "Updated body content"},
									},
								},
							},
						},
					},
				},
			},
			expected: []*action{
				// Update first slide (exact layout and title match)
				{
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
					slide: &Slide{
						Layout:    "TITLE",
						Titles:    []string{"Target Title"},
						Subtitles: []string{"New subtitle"},
					},
				},
				// Update second slide (exact layout and title match)
				{
					actionType:  actionTypeUpdate,
					index:       1,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE_AND_BODY",
						Titles: []string{"Target Title"},
						Bodies: []*Body{
							{
								Paragraphs: []*Paragraph{
									{
										Bullet: BulletNone,
										Fragments: []*Fragment{
											{Value: "Updated body content"},
										},
									},
								},
							},
						},
					},
				},
				// Delete action for unused slide
				{
					actionType:  actionTypeDelete,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Other Title"},
					},
				},
			},
		},
		{
			name: "prefer exact layout and title match over index order",
			before: Slides{
				{
					Layout: "TITLE_AND_BODY",
					Titles: []string{"Different Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Bullet: BulletNone,
									Fragments: []*Fragment{
										{Value: "Some content"},
									},
								},
							},
						},
					},
				},
				{
					Layout:    "TITLE",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"Old subtitle"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Another Title"},
				},
			},
			after: Slides{
				{
					Layout:    "TITLE",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"New subtitle"},
				},
			},
			expected: []*action{
				// Should move slide at index 1 (exact layout and title match) to index 0
				// then update the content
				{
					actionType:  actionTypeMove,
					index:       1,
					moveToIndex: 0,
					slide: &Slide{
						Layout:    "TITLE",
						Titles:    []string{"Target Title"},
						Subtitles: []string{"Old subtitle"},
					},
				},
				{
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
					slide: &Slide{
						Layout:    "TITLE",
						Titles:    []string{"Target Title"},
						Subtitles: []string{"New subtitle"},
					},
				},
				// Delete actions for unused slides
				{
					actionType:  actionTypeDelete,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Another Title"},
					},
				},
			},
		},
		{
			name: "update and multiple delete",
			before: Slides{
				{
					Layout:    "TITLE",
					Titles:    []string{"Title"},
					Subtitles: []string{"subtitle"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Title 2"},
				},
				{
					Layout: "TITLE",
					Titles: []string{"Title 3"},
				},
			},
			after: Slides{
				{
					Layout:    "TITLE_AND_BODY",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"subtitle"},
				},
			},
			expected: []*action{
				{
					actionType:  actionTypeUpdate,
					index:       0,
					moveToIndex: -1,
					slide: &Slide{
						Layout:    "TITLE_AND_BODY",
						Titles:    []string{"Target Title"},
						Subtitles: []string{"subtitle"},
					},
				},
				{
					actionType:  actionTypeDelete,
					index:       2,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Title 3"},
					},
				},
				{
					actionType:  actionTypeDelete,
					index:       1,
					moveToIndex: -1,
					slide: &Slide{
						Layout: "TITLE",
						Titles: []string{"Title 2"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actions []*action
			var err error

			actions, err = diffSlides(tt.before, tt.after)

			if err != nil {
				t.Fatalf("diffSlides() error = %v", err)
			}

			if len(actions) != len(tt.expected) {
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

func TestGetSimilarityPriority(t *testing.T) {
	tests := []struct {
		name     string
		slide1   *Slide
		slide2   *Slide
		expected int
	}{
		{
			name:     "nil slides",
			slide1:   nil,
			slide2:   nil,
			expected: 6,
		},
		{
			name:   "one nil slide",
			slide1: nil,
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Test"},
			},
			expected: 6,
		},
		{
			name: "exact layout and title match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title"},
			},
			expected: 2,
		},
		{
			name: "title match only",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Same Title"},
			},
			expected: 4,
		},
		{
			name: "layout match only",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title 2"},
			},
			expected: 5,
		},
		{
			name: "no match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Title 2"},
			},
			expected: 7, // No match
		},
		{
			name: "layout match with no titles",
			slide1: &Slide{
				Layout: "TITLE",
			},
			slide2: &Slide{
				Layout: "TITLE",
			},
			expected: 2, // Layout and title match (both have empty titles)
		},
		{
			name: "subtitle match only",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 6, // Subtitle match only
		},
		{
			name: "layout and subtitle match",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 3, // Layout and subtitle match
		},
		{
			name: "multiple titles - exact match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Title A", "Title B"},
			},
			expected: 4, // Title match only (all titles match exactly)
		},
		{
			name: "multiple titles - partial match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title", "Different Title"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Same Title", "Another Title"},
			},
			expected: 7, // No match (not all titles match)
		},
		{
			name: "multiple titles - different order",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{"Title B", "Title A"},
			},
			expected: 7, // No match (order matters for exact match)
		},
		{
			name: "multiple subtitles - exact match",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			expected: 6, // Subtitle match only (all subtitles match exactly)
		},
		{
			name: "multiple subtitles - partial match",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle", "Different Subtitle"},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle", "Another Subtitle"},
			},
			expected: 7, // No match (not all subtitles match)
		},
		{
			name: "multiple subtitles - different order",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Subtitle B", "Subtitle A"},
			},
			expected: 7, // No match (order matters for exact match)
		},
		{
			name: "layout and multiple titles exact match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Title A", "Title B"},
			},
			expected: 2, // High priority: both layout and all titles match exactly
		},
		{
			name: "layout match but titles don't match exactly",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{"Different Title", "Same Title"},
			},
			slide2: &Slide{
				Layout: "TITLE",
				Titles: []string{"Same Title", "Another Title"},
			},
			expected: 5, // Layout match only (titles don't match exactly)
		},
		{
			name: "no title or subtitle match with multiple values",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title A", "Title B"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title C", "Title D"},
				Subtitles: []string{"Subtitle C", "Subtitle D"},
			},
			expected: 7,
		},
		{
			name: "empty titles match",
			slide1: &Slide{
				Layout: "TITLE",
				Titles: []string{},
			},
			slide2: &Slide{
				Layout: "TITLE_AND_BODY",
				Titles: []string{},
			},
			expected: 4, // Title match (both have empty titles)
		},
		{
			name: "empty subtitles match",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Title 1"},
				Subtitles: []string{},
			},
			slide2: &Slide{
				Layout:    "TITLE_AND_BODY",
				Titles:    []string{"Title 2"},
				Subtitles: []string{},
			},
			expected: 7, // No match (both have empty subtitles, but no actual subtitle content)
		},
		{
			name: "layout, title, and subtitle all match",
			slide1: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Same Title"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "TITLE",
				Titles:    []string{"Same Title"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 1, // Highest priority: layout, title, and subtitle all match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSimilarityPriority(tt.slide1, tt.slide2)
			if result != tt.expected {
				t.Errorf("getSimilarityPriority() = %v, expected %v", result, tt.expected)
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
	if actual.moveToIndex != expected.moveToIndex {
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
