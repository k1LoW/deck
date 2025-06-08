package deck

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var tests = []struct {
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
				Layout: "title",
				Titles: []string{"New Slide"},
			},
		},
		expected: []*action{
			{
				actionType:  actionTypeAppend,
				index:       0,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New Slide"},
				},
			},
		},
	},
	{
		name: "delete slide",
		before: Slides{
			{
				Layout: "title",
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
					Layout: "title",
					Titles: []string{"Old Slide"},
				},
			},
		},
	},
	{
		name: "move slide",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Slide A"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide B"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"Slide B"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide A"},
			},
		},
		expected: []*action{
			{
				actionType:  actionTypeMove,
				index:       1,
				moveToIndex: 0,
				slide: &Slide{
					Layout: "title",
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
				Layout: "title",
				Titles: []string{"Original Title"},
			},
		},
		after: Slides{
			{
				Layout:    "title",
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
					Layout:    "title",
					Titles:    []string{"Original Title"},
					Subtitles: []string{"New Subtitle"},
				},
			},
		},
	},
	{
		name: "move slide and update content",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Slide 1"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 2"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 3"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"Slide 2"},
			},
			{
				Layout: "title",
				Titles: []string{"New Slide"},
			},
			{
				Layout:    "title",
				Titles:    []string{"Slide 1"},
				Subtitles: []string{"Updated"},
			},
		},
		expected: []*action{
			// Move Slide 2 to position 0
			{
				actionType:  actionTypeMove,
				index:       1,
				moveToIndex: 0,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Slide 2"},
				},
			},
			// Update actions (New Slide replaces existing slide at index 1)
			{
				actionType:  actionTypeUpdate,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New Slide"},
				},
			},
			// Update actions (Slide 1 with subtitles is detected as update at index 2)
			{
				actionType:  actionTypeUpdate,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title",
					Titles:    []string{"Slide 1"},
					Subtitles: []string{"Updated"},
				},
			},
		},
	},
	{
		name: "update slides and delete unused ones",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Slide 1"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 2"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 3"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 4"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 5"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"New Slide A"},
			},
			{
				Layout: "title",
				Titles: []string{"Slide 2"},
			},
			{
				Layout: "title",
				Titles: []string{"New Slide B"},
			},
			{
				Layout: "title",
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
					Layout: "title",
					Titles: []string{"New Slide A"},
				},
			},
			{
				actionType:  actionTypeUpdate,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New Slide B"},
				},
			},
			// Delete actions for unused slides
			{
				actionType:  actionTypeDelete,
				index:       4,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Slide 5"},
				},
			},
		},
	},
	{
		name: "move slides and delete unused one",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"B"},
			},
			{
				Layout: "title",
				Titles: []string{"C"},
			},
			{
				Layout: "title",
				Titles: []string{"D"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"D"}, // moved from index 3 to 0
			},
			{
				Layout: "title",
				Titles: []string{"B"}, // moved from index 1 to 1 (no change)
			},
			{
				Layout: "title",
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
					Layout: "title",
					Titles: []string{"D"},
				},
			},
			{
				actionType:  actionTypeMove,
				index:       2,
				moveToIndex: 1, // B moved after D's move
				slide: &Slide{
					Layout: "title",
					Titles: []string{"B"},
				},
			},
			// Delete action for unused slide
			{
				actionType:  actionTypeDelete,
				index:       3,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"C"},
				},
			},
		},
	},
	{
		name: "update first slide content only",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"First Page"}, // This will be updated
			},
			{
				Layout: "title",
				Titles: []string{"Second Page"},
			},
			{
				Layout: "title",
				Titles: []string{"Third Page"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"New First Page"}, // Updated page at index 0
			},
			{
				Layout: "title",
				Titles: []string{"Second Page"}, // No change
			},
			{
				Layout: "title",
				Titles: []string{"Third Page"}, // No change
			},
		},
		expected: []*action{
			{
				actionType:  actionTypeUpdate,
				index:       0,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New First Page"},
				},
			},
		},
	},
	{
		name: "move slides and update existing page",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"First Page"}, // This will be replaced
			},
			{
				Layout: "title",
				Titles: []string{"Second Page"}, // This will move to index 1
			},
			{
				Layout: "title",
				Titles: []string{"Third Page"}, // This will move to index 0
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"Third Page"}, // Moved from index 2 to 0
			},
			{
				Layout: "title",
				Titles: []string{"Second Page"}, // Moved from index 1 to 1
			},
			{
				Layout: "title",
				Titles: []string{"New Page"}, // Added at index 2
			},
		},
		expected: []*action{
			// Move Third Page to position 0
			{
				actionType:  actionTypeMove,
				index:       2,
				moveToIndex: 0,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Third Page"},
				},
			},
			// Move Second Page to position 1
			{
				actionType:  actionTypeMove,
				index:       2,
				moveToIndex: 1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Second Page"},
				},
			},
			// Update existing page at index 2
			{
				actionType:  actionTypeUpdate,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New Page"},
				},
			},
		},
	},
	{
		name: "move slides, update page, and delete unused slide",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Delete Me 1"}, // index 0 - will be replaced
			},
			{
				Layout: "title",
				Titles: []string{"Delete Me 2"}, // index 1 - will be replaced
			},
			{
				Layout: "title",
				Titles: []string{"Keep Me A"}, // index 2 - will move to index 1
			},
			{
				Layout: "title",
				Titles: []string{"Keep Me B"}, // index 3 - will move to index 0
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"Keep Me B"}, // moved from index 3 to 0
			},
			{
				Layout: "title",
				Titles: []string{"Keep Me A"}, // moved from index 2 to 1
			},
			{
				Layout: "title",
				Titles: []string{"New Page"}, // added at index 2
			},
		},
		expected: []*action{
			// Move Keep Me B to position 0
			{
				actionType:  actionTypeMove,
				index:       3,
				moveToIndex: 0,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Keep Me B"},
				},
			},
			// Move Keep Me A to position 1
			{
				actionType:  actionTypeMove,
				index:       3,
				moveToIndex: 1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Keep Me A"},
				},
			},
			// Update existing page at index 2
			{
				actionType:  actionTypeUpdate,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"New Page"},
				},
			},
			// Delete actions for unused slides
			{
				actionType:  actionTypeDelete,
				index:       3,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Delete Me 2"},
				},
			},
		},
	},
	{
		name: "reuse slide with same layout and title after move",
		before: Slides{
			{
				Layout: "title",
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
				Layout: "title-and-body",
				Titles: []string{"Different Title"},
			},
			{
				Layout: "title",
				Titles: []string{"Another Title"},
			},
		},
		after: Slides{
			{
				Layout: "title-and-body",
				Titles: []string{"Different Title"},
			},
			{
				Layout: "title",
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
			{
				actionType:  actionTypeMove,
				index:       1,
				moveToIndex: 0,
				slide: &Slide{
					Layout: "title-and-body",
					Titles: []string{"Different Title"},
				},
			},
			{
				actionType:  actionTypeUpdate,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
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
			{
				actionType:  actionTypeDelete,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Another Title"},
				},
			},
		},
	},
	{
		name: "prioritize exact layout and title match for reuse",
		before: Slides{
			{
				Layout:    "title",
				Titles:    []string{"Target Title"},
				Subtitles: []string{"Old subtitle"},
			},
			{
				Layout: "title-and-body",
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
				Layout: "title",
				Titles: []string{"Other Title"},
			},
		},
		after: Slides{
			{
				Layout:    "title",
				Titles:    []string{"Target Title"},
				Subtitles: []string{"New subtitle"},
			},
			{
				Layout: "title-and-body",
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
					Layout:    "title",
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
					Layout: "title-and-body",
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
					Layout: "title",
					Titles: []string{"Other Title"},
				},
			},
		},
	},
	{
		name: "prefer exact layout and title match over index order",
		before: Slides{
			{
				Layout: "title-and-body",
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
				Layout:    "title",
				Titles:    []string{"Target Title"},
				Subtitles: []string{"Old subtitle"},
			},
			{
				Layout: "title",
				Titles: []string{"Another Title"},
			},
		},
		after: Slides{
			{
				Layout:    "title",
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
					Layout:    "title",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"Old subtitle"},
				},
			},
			{
				actionType:  actionTypeUpdate,
				index:       0,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"New subtitle"},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Another Title"},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title-and-body",
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
			},
		},
	},
	{
		name: "update and multiple delete",
		before: Slides{
			{
				Layout:    "title",
				Titles:    []string{"Title"},
				Subtitles: []string{"subtitle"},
			},
			{
				Layout: "title",
				Titles: []string{"Title 2"},
			},
			{
				Layout: "title",
				Titles: []string{"Title 3"},
			},
		},
		after: Slides{
			{
				Layout:    "title-and-body",
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
					Layout:    "title-and-body",
					Titles:    []string{"Target Title"},
					Subtitles: []string{"subtitle"},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Title 3"},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"Title 2"},
				},
			},
		},
	},
	{
		name: "duplicate slides reordering - A A B A to A B A A",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 0
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 1
			{
				Layout: "title",
				Titles: []string{"B"},
			}, // index 2
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 3
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 0 (no change)
			{
				Layout: "title",
				Titles: []string{"B"},
			}, // index 1 (moved from index 2)
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 2 (moved from index 1)
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 3 (no change)
		},
		expected: []*action{
			{
				actionType:  actionTypeMove,
				index:       2,
				moveToIndex: 1,
				slide: &Slide{
					Layout: "title",
					Titles: []string{"B"},
				},
			},
		},
	},
	{
		name: "move slide to correct position and delete unused slide",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body-half",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body-3col",
				Titles: []string{"A"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body-3col",
				Titles: []string{"A"},
			},
		},
		expected: []*action{
			{
				actionType:  actionTypeMove,
				index:       3,
				moveToIndex: 2,
				slide: &Slide{
					Layout: "title-and-body-3col",
					Titles: []string{"A"},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       3,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title-and-body-half",
					Titles: []string{"A"},
				},
			},
		},
	},
	{
		name: "delete excess slides when after is shorter than before",
		before: Slides{
			{
				Layout:    "title-and-body-3col",
				Titles:    []string{"CAP theorem"},
				Subtitles: []string{"In Database theory", "Consistency", "Availability", "Partition tolerance"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every read receives the most recent write or an error."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every request received by a non-failing node in the system must result in a response."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "The system continues to operate despite an arbitrary number of messages being dropped."},
								},
							},
						},
					},
				},
			},
			{
				Layout:    "title",
				Titles:    []string{"Title"},
				Subtitles: []string{"Subtitle"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Body content"},
								},
							},
						},
					},
				},
			},
			{
				Layout:      "section",
				Titles:      []string{"Title"},
				SpeakerNote: "comment\n\ncomment",
			},
			{
				Layout: "title-and-body",
				Titles: []string{"Title"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Body content"},
								},
							},
						},
					},
				},
			},
			{
				Layout:    "title-and-body-3col",
				Titles:    []string{"1"},
				Subtitles: []string{"2", "3", "4", "5"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Body 1"},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Body 2"},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Body 3"},
								},
							},
						},
					},
				},
			},
		},
		after: Slides{
			{
				Layout:    "title-and-body-3col",
				Titles:    []string{"CAP theorem"},
				Subtitles: []string{"In Database theory", "Consistency", "Availability", "Partition tolerance"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every read receives the most recent write or an error."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every request received by a non-failing node in the system must result in a response."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "The system continues to operate despite an arbitrary number of messages being dropped."},
								},
							},
						},
					},
				},
			},
		},
		expected: []*action{
			// Delete actions for the 4 excess slides (indices 4, 3, 2, 1 in descending order)
			{
				actionType:  actionTypeDelete,
				index:       4,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title-and-body-3col",
					Titles:    []string{"1"},
					Subtitles: []string{"2", "3", "4", "5"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Body 1"},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Body 2"},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Body 3"},
									},
								},
							},
						},
					},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       3,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title-and-body",
					Titles: []string{"Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Body content"},
									},
								},
							},
						},
					},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout:      "section",
					Titles:      []string{"Title"},
					SpeakerNote: "comment\n\ncomment",
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title",
					Titles:    []string{"Title"},
					Subtitles: []string{"Subtitle"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Body content"},
									},
								},
							},
						},
					},
				},
			},
		},
	},
	{
		name: "integration test scenario - slide.md then cap.md sequence",
		before: Slides{
			{
				Layout:    "title",
				Titles:    []string{"Title"},
				Subtitles: []string{"Subtitle"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "@k1LoW"},
								},
							},
						},
					},
				},
			},
			{
				Layout:      "section",
				Titles:      []string{"Title"},
				SpeakerNote: "comment\n\ncomment",
			},
			{
				Layout: "title-and-body",
				Titles: []string{"Title"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "aA"},
								},
								Bullet: BulletDash,
							},
							{
								Fragments: []*Fragment{
									{Value: "b", Bold: true},
									{Value: "B"},
								},
								Bullet: BulletDash,
							},
							{
								Fragments: []*Fragment{
									{Value: "cC"},
								},
								Bullet: BulletDash,
							},
						},
					},
				},
			},
			{
				Layout: "title-and-body",
				Titles: []string{"1"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "body"},
								},
							},
						},
					},
				},
			},
			{
				Layout:    "title-and-body-3col",
				Titles:    []string{"1"},
				Subtitles: []string{"2", "3", "4", "5"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "a"},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "b"},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "c"},
								},
							},
						},
					},
				},
			},
		},
		after: Slides{
			// Simulating cap.md test (1 slide only)
			{
				Layout:    "title-and-body-3col",
				Titles:    []string{"CAP theorem"},
				Subtitles: []string{"In Database theory", "Consistency", "Availability", "Partition tolerance"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every read receives the most recent write or an error."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Every request received by a non-failing node in the system must result in a response."},
								},
							},
						},
					},
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "The system continues to operate despite an arbitrary number of messages being dropped."},
								},
							},
						},
					},
				},
			},
		},
		expected: []*action{
			// Update the first slide to CAP theorem (reuse slide at index 4 which has same layout)
			{
				actionType:  actionTypeMove,
				index:       4,
				moveToIndex: 0,
				slide: &Slide{
					Layout:    "title-and-body-3col",
					Titles:    []string{"1"},
					Subtitles: []string{"2", "3", "4", "5"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "a"},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "b"},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "c"},
									},
								},
							},
						},
					},
				},
			},
			{
				actionType:  actionTypeUpdate,
				index:       0,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title-and-body-3col",
					Titles:    []string{"CAP theorem"},
					Subtitles: []string{"In Database theory", "Consistency", "Availability", "Partition tolerance"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Every read receives the most recent write or an error."},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Every request received by a non-failing node in the system must result in a response."},
									},
								},
							},
						},
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "The system continues to operate despite an arbitrary number of messages being dropped."},
									},
								},
							},
						},
					},
				},
			},
			// Delete the remaining 4 slides (indices 4, 3, 2, 1 in descending order after move)
			{
				actionType:  actionTypeDelete,
				index:       4,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title-and-body",
					Titles: []string{"1"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "body"},
									},
								},
							},
						},
					},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       3,
				moveToIndex: -1,
				slide: &Slide{
					Layout: "title-and-body",
					Titles: []string{"Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "aA"},
									},
									Bullet: BulletDash,
								},
								{
									Fragments: []*Fragment{
										{Value: "b", Bold: true},
										{Value: "B"},
									},
									Bullet: BulletDash,
								},
								{
									Fragments: []*Fragment{
										{Value: "cC"},
									},
									Bullet: BulletDash,
								},
							},
						},
					},
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       2,
				moveToIndex: -1,
				slide: &Slide{
					Layout:      "section",
					Titles:      []string{"Title"},
					SpeakerNote: "comment\n\ncomment",
				},
			},
			{
				actionType:  actionTypeDelete,
				index:       1,
				moveToIndex: -1,
				slide: &Slide{
					Layout:    "title",
					Titles:    []string{"Title"},
					Subtitles: []string{"Subtitle"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "@k1LoW"},
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

func TestDiffSlides(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actions []*action
			var err error

			actions, err = diffSlides(tt.before, tt.after)

			if err != nil {
				t.Fatalf("diffSlides() error = %v", err)
			}

			if len(actions) != len(tt.expected) {
				t.Logf("Actual actions:")
				for i, action := range actions {
					t.Logf("  [%d] %s: index=%d, moveToIndex=%d, slide=%+v", i, action.actionType, action.index, action.moveToIndex, action.slide)
				}
				t.Logf("Expected actions:")
				for i, action := range tt.expected {
					t.Logf("  [%d] %s: index=%d, moveToIndex=%d, slide=%+v", i, action.actionType, action.index, action.moveToIndex, action.slide)
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

func TestApply(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	ctx := context.Background()
	presentationID := os.Getenv("TEST_PRESENTATION_ID")
	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := New(ctx, WithPresentationID(presentationID))
			if err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, tt.before); err != nil {
				t.Fatal(err)
			}
			if _, err := d.DumpSlides(ctx); err != nil {
				t.Fatal(err)
			}
			if err := d.Apply(ctx, tt.after); err != nil {
				t.Fatal(err)
			}
			after, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.after, after, cmpopts...); diff != "" {
				t.Errorf("diff after apply: %s", diff)
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
			expected: 7,
		},
		{
			name:   "one nil slide",
			slide1: nil,
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Test"},
			},
			expected: 7,
		},
		{
			name: "exact layout and title match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			expected: 0,
		},
		{
			name: "title match only",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{"Same Title"},
			},
			expected: 4,
		},
		{
			name: "layout match only",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title 2"},
			},
			expected: 5,
		},
		{
			name: "no match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{"Title 2"},
			},
			expected: 7, // No match
		},
		{
			name: "layout match with no titles",
			slide1: &Slide{
				Layout: "title",
			},
			slide2: &Slide{
				Layout: "title",
			},
			expected: 0, // Perfect match (both have same layout and empty titles)
		},
		{
			name: "subtitle match only",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 6, // Subtitle match only
		},
		{
			name: "layout and subtitle match",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 3, // Layout and subtitle match
		},
		{
			name: "multiple titles - exact match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{"Title A", "Title B"},
			},
			expected: 4, // Title match only (all titles match exactly)
		},
		{
			name: "multiple titles - partial match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Same Title", "Different Title"},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{"Same Title", "Another Title"},
			},
			expected: 7, // No match (not all titles match)
		},
		{
			name: "multiple titles - different order",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{"Title B", "Title A"},
			},
			expected: 7, // No match (order matters for exact match)
		},
		{
			name: "multiple subtitles - exact match",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			expected: 6, // Subtitle match only (all subtitles match exactly)
		},
		{
			name: "multiple subtitles - partial match",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Same Subtitle", "Different Subtitle"},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Same Subtitle", "Another Subtitle"},
			},
			expected: 7, // No match (not all subtitles match)
		},
		{
			name: "multiple subtitles - different order",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title 2"},
				Subtitles: []string{"Subtitle B", "Subtitle A"},
			},
			expected: 7, // No match (order matters for exact match)
		},
		{
			name: "layout and multiple titles exact match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title A", "Title B"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title A", "Title B"},
			},
			expected: 0, // Perfect match: both layout and all titles match exactly
		},
		{
			name: "layout match but titles don't match exactly",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Different Title", "Same Title"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Same Title", "Another Title"},
			},
			expected: 5, // Layout match only (titles don't match exactly)
		},
		{
			name: "no title or subtitle match with multiple values",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title A", "Title B"},
				Subtitles: []string{"Subtitle A", "Subtitle B"},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title C", "Title D"},
				Subtitles: []string{"Subtitle C", "Subtitle D"},
			},
			expected: 7,
		},
		{
			name: "empty titles match",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{},
			},
			slide2: &Slide{
				Layout: "title-and-body",
				Titles: []string{},
			},
			expected: 4, // Title match (both have empty titles)
		},
		{
			name: "empty subtitles match",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Title 1"},
				Subtitles: []string{},
			},
			slide2: &Slide{
				Layout:    "title-and-body",
				Titles:    []string{"Title 2"},
				Subtitles: []string{},
			},
			expected: 7, // No match (both have empty subtitles, but no actual subtitle content)
		},
		{
			name: "layout, title, and subtitle all match",
			slide1: &Slide{
				Layout:    "title",
				Titles:    []string{"Same Title"},
				Subtitles: []string{"Same Subtitle"},
			},
			slide2: &Slide{
				Layout:    "title",
				Titles:    []string{"Same Title"},
				Subtitles: []string{"Same Subtitle"},
			},
			expected: 0, // Perfect match: layout, title, and subtitle all match
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

	return slide1.SpeakerNote == slide2.SpeakerNote
}
