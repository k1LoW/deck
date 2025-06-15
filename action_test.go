package deck

import (
	"maps"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var tests = []struct {
	name   string
	before Slides
	after  Slides
}{
	{
		name:   "empty slides",
		before: Slides{},
		after:  Slides{},
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
	},
	{
		name: "duplicate slides reordering - A B A A to A A B A",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 0
			{
				Layout: "title",
				Titles: []string{"B"},
			}, // index 1
			{
				Layout: "title",
				Titles: []string{"A"},
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
				Titles: []string{"A"},
			}, // index 1 (corrected: "B" → "A")
			{
				Layout: "title",
				Titles: []string{"B"},
			}, // index 2 (corrected: "A" → "B")
			{
				Layout: "title",
				Titles: []string{"A"},
			}, // index 3 (no change)
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
	},
	{
		name: "prefer move over update when better match exists elsewhere",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Different Title"},
			},
			{
				Layout: "title-and-body",
				Titles: []string{"Target Title"},
			},
		},
		after: Slides{
			{
				Layout: "title-and-body",
				Titles: []string{"Target Title"},
			},
		},
	},
	{
		name: "prefer move over update with layout and subtitle match",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Different Title"},
			},
			{
				Layout:    "title-and-body",
				Titles:    []string{"Another Title"},
				Subtitles: []string{"Same Subtitle"},
			},
		},
		after: Slides{
			{
				Layout:    "title-and-body",
				Titles:    []string{"New Title"},
				Subtitles: []string{"Same Subtitle"},
			},
		},
	},
	{
		name: "insert slide with reuse - A B C to A D B C",
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
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"D"},
			},
			{
				Layout:    "title",
				Titles:    []string{"B"},
				Subtitles: []string{"Subtitle B"},
			},
			{
				Layout:    "title",
				Titles:    []string{"C"},
				Subtitles: []string{"Subtitle C"},
			},
		},
	},
	{
		name: "insert slide without reuse - similarity > 3",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title-and-body", // Different layout from target
				Titles: []string{"B"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"D"},
			},
			{
				Layout: "title", // Different layout from before - similarity = 4 (title match only)
				Titles: []string{"B"},
			},
		},
	},
	{
		name: "insert slide with mixed similarity - some reuse, some not",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"B"}, // similarity = 2 (layout + title match) <= 3
			},
			{
				Layout: "title-and-body",
				Titles: []string{"C"}, // similarity = 4 (title match only) > 3
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"D"},
			},
			{
				Layout: "title",
				Titles: []string{"B"}, // Should be moved (similarity <= 3)
			},
			{
				Layout: "title",       // Different layout from before - similarity = 4 (title match only) > 3
				Titles: []string{"C"}, // Should not be moved (similarity > 3)
			},
		},
	},
	{
		name: "reorder slides - simple swap with reuse (NEW)",
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
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"B"},
			},
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"C"},
			},
		},
	},
	{
		name: "delete and reorder with reuse (NEW)",
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
				Titles: []string{"C"},
			},
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"D"},
			},
		},
	},
	{
		name: "complete reverse order with reuse (NEW)",
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
				Titles: []string{"D"},
			},
			{
				Layout: "title",
				Titles: []string{"C"},
			},
			{
				Layout: "title",
				Titles: []string{"B"},
			},
			{
				Layout: "title",
				Titles: []string{"A"},
			},
		},
	},
	{
		name: "split one slide into three slides - no similarity",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Original Slide"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"First Part"},
			},
			{
				Layout: "title",
				Titles: []string{"Second Part"},
			},
			{
				Layout: "title",
				Titles: []string{"Third Part"},
			},
		},
	},
	{
		name: "split one slide into three slides - first slide has title similarity",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Shared Title"},
			},
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"Shared Title"},
			},
			{
				Layout: "title",
				Titles: []string{"New Title 1"},
			},
			{
				Layout: "title",
				Titles: []string{"New Title 2"},
			},
		},
	},
	{
		name: "split one slide into three slides - mixed layouts",
		before: Slides{
			{
				Layout: "title",
				Titles: []string{"Original Content"},
			},
		},
		after: Slides{
			{
				Layout: "title-and-body",
				Titles: []string{"Section 1"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Content for section 1"},
								},
							},
						},
					},
				},
			},
			{
				Layout: "title",
				Titles: []string{"Section 2"},
			},
			{
				Layout: "title-and-body",
				Titles: []string{"Section 3"},
				Bodies: []*Body{
					{
						Paragraphs: []*Paragraph{
							{
								Fragments: []*Fragment{
									{Value: "Content for section 3"},
								},
							},
						},
					},
				},
			},
		},
	},
	{
		name: "complex reordering with insertions - A B C to D B A E F",
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
		},
		after: Slides{
			{
				Layout: "title",
				Titles: []string{"D"},
			},
			{
				Layout: "title",
				Titles: []string{"B"},
			},
			{
				Layout: "title",
				Titles: []string{"A"},
			},
			{
				Layout: "title",
				Titles: []string{"E"},
			},
			{
				Layout: "title",
				Titles: []string{"F"},
			},
		},
	},
	{
		name: "complex reordering with new slides and content updates",
		before: Slides{
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
		after: Slides{
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
	},
}

func TestGenerateActions(t *testing.T) {
	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions, err := generateActions(tt.before, tt.after)
			if err != nil {
				t.Fatal(err)
			}
			got := actionsEmulator(t, tt.before, actions)
			if diff := cmp.Diff(got, tt.after, cmpopts...); diff != "" {
				t.Error(diff)
			}
		})
	}
}

// TestAdjustSlideCount tests the adjustSlideCount function.
func TestAdjustSlideCount(t *testing.T) {
	tests := []struct {
		name           string
		before         Slides
		after          Slides
		expectedBefore Slides
		expectedAfter  Slides
	}{
		{
			name: "same count - no adjustment needed",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"D"}},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"D"}},
			},
		},
		{
			name: "after is shorter - add slides to after with .delete = true",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}}, // high similarity with before[0]
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}, delete: true}, // lowest similarity score
				{Layout: "title", Titles: []string{"C"}, delete: true}, // second lowest similarity score
			},
		},
		{
			name: "before is shorter - add slides to before with .new = true",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}}, // high similarity with after[0]
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}, new: true}, // lowest similarity score
				{Layout: "title", Titles: []string{"C"}, new: true}, // second lowest similarity score
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
		},
		{
			name:   "empty before - add all after slides to before with .new = true",
			before: Slides{},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}, new: true},
				{Layout: "title", Titles: []string{"B"}, new: true},
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
		},
		{
			name: "empty after - add all before slides to after with .delete = true",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"A"}, delete: true},
				{Layout: "title", Titles: []string{"B"}, delete: true},
			},
		},
		{
			name: "complex similarity calculation - layout and content differences",
			before: Slides{
				{
					Layout: "title",
					Titles: []string{"Same Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Content A"},
									},
								},
							},
						},
					},
				}, // High similarity with after[0]
				{Layout: "title-and-body", Titles: []string{"Different Title"}}, // Low similarity
				{Layout: "title", Titles: []string{"Another Title"}},            // Medium similarity
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Same Title"}}, // High similarity with before[0]
			},
			expectedBefore: Slides{
				{
					Layout: "title",
					Titles: []string{"Same Title"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Content A"},
									},
								},
							},
						},
					},
				},
				{Layout: "title-and-body", Titles: []string{"Different Title"}},
				{Layout: "title", Titles: []string{"Another Title"}},
			},
			expectedAfter: Slides{
				{Layout: "title", Titles: []string{"Same Title"}},
				{Layout: "title-and-body", Titles: []string{"Different Title"}, delete: true}, // Lowest similarity
				{Layout: "title", Titles: []string{"Another Title"}, delete: true},            // Second lowest
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adjustedBefore, adjustedAfter, err := adjustSlideCount(tt.before, tt.after)
			if err != nil {
				t.Fatalf("adjustSlideCount() error = %v", err)
			}

			// Check lengths are equal
			if len(adjustedBefore) != len(adjustedAfter) {
				t.Errorf("adjustSlideCount() lengths not equal: before=%d, after=%d", len(adjustedBefore), len(adjustedAfter))
			}

			// Compare with expected results
			cmpopts := cmp.Options{
				cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
				cmpopts.IgnoreUnexported(Slide{}),
			}

			if diff := cmp.Diff(tt.expectedBefore, adjustedBefore, cmpopts...); diff != "" {
				t.Errorf("adjustSlideCount() before mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(tt.expectedAfter, adjustedAfter, cmpopts...); diff != "" {
				t.Errorf("adjustSlideCount() after mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestMapSlides tests the mapSlides function.
func TestMapSlides(t *testing.T) {
	tests := []struct {
		name     string
		before   Slides
		after    Slides
		expected map[int]int
	}{
		{
			name: "basic perfect match",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expected: map[int]int{0: 0, 1: 1},
		},
		{
			name: "simple swap",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
			expected: map[int]int{0: 1, 1: 0},
		},
		{
			name: "layout and title priority",
			before: Slides{
				{Layout: "title", Titles: []string{"Same Title"}},
				{Layout: "title-and-body", Titles: []string{"Different Title"}},
			},
			after: Slides{
				{Layout: "title-and-body", Titles: []string{"Different Title"}},
				{Layout: "title", Titles: []string{"Same Title"}},
			},
			expected: map[int]int{0: 1, 1: 0}, // Layout+Title match should be prioritized
		},
		{
			name: "position bonus consideration",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expected: map[int]int{0: 1, 1: 2, 2: 0}, // Position bonus should influence mapping
		},
		{
			name: "complex optimization",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"D"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"D"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			expected: map[int]int{0: 2, 1: 1, 2: 3, 3: 0}, // Optimal mapping considering all factors
		},
		{
			name: "single slide",
			before: Slides{
				{Layout: "title", Titles: []string{"Only"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Only"}},
			},
			expected: map[int]int{0: 0},
		},
		{
			name: "different layouts with same titles",
			before: Slides{
				{Layout: "title", Titles: []string{"Same"}},
				{Layout: "title-and-body", Titles: []string{"Same"}},
			},
			after: Slides{
				{Layout: "title-and-body", Titles: []string{"Same"}},
				{Layout: "title", Titles: []string{"Same"}},
			},
			expected: map[int]int{0: 1, 1: 0}, // Layout match should be prioritized
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure slides are same length (prerequisite for mapSlides)
			if len(tt.before) != len(tt.after) {
				t.Fatalf("Test setup error: before and after must have same length")
			}

			result, err := mapSlides(tt.before, tt.after)
			if err != nil {
				t.Fatalf("mapSlides() error = %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("mapSlides() result length = %d, expected %d", len(result), len(tt.expected))
			}

			for beforeIdx, expectedAfterIdx := range tt.expected {
				if actualAfterIdx, ok := result[beforeIdx]; !ok {
					t.Errorf("mapSlides() missing mapping for before index %d", beforeIdx)
				} else if actualAfterIdx != expectedAfterIdx {
					t.Errorf("mapSlides() mapping[%d] = %d, expected %d", beforeIdx, actualAfterIdx, expectedAfterIdx)
				}
			}

			// Verify that all after indices are used exactly once
			usedAfterIndices := make(map[int]bool)
			for _, afterIdx := range result {
				if usedAfterIndices[afterIdx] {
					t.Errorf("mapSlides() after index %d used multiple times", afterIdx)
				}
				usedAfterIndices[afterIdx] = true
			}

			// Verify all after indices from 0 to len-1 are used
			for i := range len(tt.after) {
				if !usedAfterIndices[i] {
					t.Errorf("mapSlides() after index %d not used", i)
				}
			}
		})
	}
}

// TestMapSlidesErrors tests error cases for mapSlides function.
func TestMapSlidesErrors(t *testing.T) {
	tests := []struct {
		name    string
		before  Slides
		after   Slides
		wantErr bool
	}{
		{
			name: "different lengths",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			wantErr: true, // Expect error for different lengths
		},
		{
			name:    "empty slides",
			before:  Slides{},
			after:   Slides{},
			wantErr: false, // No error expected for empty slides
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mapSlides(tt.before, tt.after)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapSlides() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestApplyDeleteMarks tests the applyDeleteMarks function.
func TestApplyDeleteMarks(t *testing.T) {
	tests := []struct {
		name            string
		before          Slides
		after           Slides
		mapping         map[int]int
		expectedDeleted []bool // before[i].delete の期待値
	}{
		{
			name: "no deleted slides",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			mapping:         map[int]int{0: 0, 1: 1},
			expectedDeleted: []bool{false, false},
		},
		{
			name: "single deleted slide",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}, delete: true},
			},
			mapping:         map[int]int{0: 0, 1: 1},
			expectedDeleted: []bool{false, true},
		},
		{
			name: "multiple deleted slides with reordering",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"C"}, delete: true},
				{Layout: "title", Titles: []string{"A"}, delete: true},
				{Layout: "title", Titles: []string{"B"}},
			},
			mapping:         map[int]int{0: 1, 1: 2, 2: 0}, // A->1, B->2, C->0
			expectedDeleted: []bool{true, false, true},     // A(deleted), B(not deleted), C(deleted)
		},
		{
			name: "all slides deleted",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"B"}, delete: true},
				{Layout: "title", Titles: []string{"A"}, delete: true},
			},
			mapping:         map[int]int{0: 1, 1: 0},
			expectedDeleted: []bool{true, true},
		},
		{
			name: "boundary check - invalid mapping indices",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}, delete: true},
			},
			mapping:         map[int]int{0: 0, 5: 10}, // 無効なインデックス
			expectedDeleted: []bool{true},
		},
		{
			name:            "empty slides",
			before:          Slides{},
			after:           Slides{},
			mapping:         map[int]int{},
			expectedDeleted: []bool{},
		},
		{
			name: "complex scenario with mixed delete flags",
			before: Slides{
				{Layout: "title", Titles: []string{"Keep1"}},
				{Layout: "title", Titles: []string{"Delete1"}},
				{Layout: "title", Titles: []string{"Keep2"}},
				{Layout: "title", Titles: []string{"Delete2"}},
				{Layout: "title", Titles: []string{"Delete3"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Delete2"}, delete: true},
				{Layout: "title", Titles: []string{"Keep1"}},
				{Layout: "title", Titles: []string{"Delete3"}, delete: true},
				{Layout: "title", Titles: []string{"Keep2"}},
				{Layout: "title", Titles: []string{"Delete1"}, delete: true},
			},
			mapping:         map[int]int{0: 1, 1: 4, 2: 3, 3: 0, 4: 2}, // Keep1->1, Delete1->4, Keep2->3, Delete2->0, Delete3->2
			expectedDeleted: []bool{false, true, false, true, true},    // Keep1(false), Delete1(true), Keep2(false), Delete2(true), Delete3(true)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyDeleteMarks(tt.before, tt.after, tt.mapping)

			if len(tt.after) != len(tt.expectedDeleted) {
				t.Fatalf("Expected %d slides, got %d", len(tt.expectedDeleted), len(tt.after))
			}

			for i, expectedDeleted := range tt.expectedDeleted {
				if tt.before[i].delete != expectedDeleted {
					t.Errorf("before[%d].delete = %v, expected %v (slide: %v)", i, tt.before[i].delete, expectedDeleted, tt.before[i].Titles)
				}
			}
		})
	}
}

// TestCopySlides tests the copySlides function.
func TestCopySlides(t *testing.T) {
	tests := []struct {
		name     string
		slides   Slides
		expected Slides
	}{
		{
			name:     "nil slides",
			slides:   nil,
			expected: nil,
		},
		{
			name:     "empty slides",
			slides:   Slides{},
			expected: Slides{},
		},
		{
			name: "single slide",
			slides: Slides{
				{
					Layout: "title",
					Titles: []string{"Test Title"},
				},
			},
			expected: Slides{
				{
					Layout: "title",
					Titles: []string{"Test Title"},
				},
			},
		},
		{
			name: "complex slide with bodies",
			slides: Slides{
				{
					Layout:    "title-and-body",
					Titles:    []string{"Complex Title"},
					Subtitles: []string{"Subtitle"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Test content", Bold: true},
									},
									Bullet: BulletDash,
								},
							},
						},
					},
					SpeakerNote: "Test note",
				},
			},
			expected: Slides{
				{
					Layout:    "title-and-body",
					Titles:    []string{"Complex Title"},
					Subtitles: []string{"Subtitle"},
					Bodies: []*Body{
						{
							Paragraphs: []*Paragraph{
								{
									Fragments: []*Fragment{
										{Value: "Test content", Bold: true},
									},
									Bullet: BulletDash,
								},
							},
						},
					},
					SpeakerNote: "Test note",
				},
			},
		},
		{
			name: "multiple slides",
			slides: Slides{
				{
					Layout: "title",
					Titles: []string{"Slide 1"},
				},
				{
					Layout: "title-and-body",
					Titles: []string{"Slide 2"},
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
			expected: Slides{
				{
					Layout: "title",
					Titles: []string{"Slide 1"},
				},
				{
					Layout: "title-and-body",
					Titles: []string{"Slide 2"},
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
	}

	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copied, err := copySlides(tt.slides)
			if err != nil {
				t.Fatalf("copySlides() error = %v", err)
			}

			// Check that the copy matches the expected result
			if diff := cmp.Diff(tt.expected, copied, cmpopts...); diff != "" {
				t.Errorf("copySlides() mismatch (-want +got):\n%s", diff)
			}

			// Check that modifying the original doesn't affect the copy
			if len(tt.slides) > 0 && len(copied) > 0 {
				originalTitle := tt.slides[0].Titles[0]
				tt.slides[0].Titles[0] = "Modified Title"

				if len(copied[0].Titles) > 0 && copied[0].Titles[0] != originalTitle {
					t.Errorf("copySlides() copy was affected by original modification")
				}

				// Restore original for other tests
				tt.slides[0].Titles[0] = originalTitle
			}
		})
	}
}

// TestDiffSlidesDoesNotModifyOriginal tests that generateActions doesn't modify the original slides.
func TestDiffSlidesDoesNotModifyOriginal(t *testing.T) {
	originalBefore := Slides{
		{
			Layout: "title",
			Titles: []string{"Original Before"},
		},
		{
			Layout: "title-and-body",
			Titles: []string{"Before Slide 2"},
			Bodies: []*Body{
				{
					Paragraphs: []*Paragraph{
						{
							Fragments: []*Fragment{
								{Value: "Original content"},
							},
						},
					},
				},
			},
		},
	}

	originalAfter := Slides{
		{
			Layout: "title",
			Titles: []string{"Original After"},
		},
	}

	// Create deep copies for comparison
	beforeCopy, err := copySlides(originalBefore)
	if err != nil {
		t.Fatalf("Failed to create before copy: %v", err)
	}

	afterCopy, err := copySlides(originalAfter)
	if err != nil {
		t.Fatalf("Failed to create after copy: %v", err)
	}

	// Execute generateActions
	_, err = generateActions(originalBefore, originalAfter)
	if err != nil {
		t.Fatalf("generateActions() error = %v", err)
	}

	// Check that original slides were not modified
	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
	}

	if diff := cmp.Diff(beforeCopy, originalBefore, cmpopts...); diff != "" {
		t.Errorf("generateActions() modified original before slides (-want +got):\n%s", diff)
	}

	if diff := cmp.Diff(afterCopy, originalAfter, cmpopts...); diff != "" {
		t.Errorf("generateActions() modified original after slides (-want +got):\n%s", diff)
	}
}

// TestGenerateDeleteActions tests the generateDeleteActions function.
func TestGenerateDeleteActions(t *testing.T) {
	tests := []struct {
		name            string
		before          Slides
		mapping         map[int]int
		expectedActions []*action
		expectedBefore  Slides      // beforeの期待される状態（削除後）
		expectedMapping map[int]int // mappingの期待される状態（削除後）
	}{
		{
			name: "no delete slides",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			mapping:         map[int]int{0: 0, 1: 1},
			expectedActions: []*action{},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expectedMapping: map[int]int{0: 0, 1: 1},
		},
		{
			name: "single delete slide at end",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}, delete: true},
			},
			mapping: map[int]int{0: 0, 1: 1},
			expectedActions: []*action{
				{
					actionType: actionTypeDelete,
					index:      1,
					slide:      &Slide{Layout: "title", Titles: []string{"B"}, delete: true},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
			},
			expectedMapping: map[int]int{0: 0},
		},
		{
			name: "multiple delete slides at end",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}, delete: true},
				{Layout: "title", Titles: []string{"D"}, delete: true},
			},
			mapping: map[int]int{0: 0, 1: 1, 2: 2, 3: 3},
			expectedActions: []*action{
				{
					actionType: actionTypeDelete,
					index:      3, // 後ろから削除するので、まずDが削除される（index 3）
					slide:      &Slide{Layout: "title", Titles: []string{"D"}, delete: true},
				},
				{
					actionType: actionTypeDelete,
					index:      2, // 次にCが削除される（index 2）
					slide:      &Slide{Layout: "title", Titles: []string{"C"}, delete: true},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			expectedMapping: map[int]int{0: 0, 1: 1},
		},
		{
			name: "delete slides with mapping adjustment",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}, delete: true},
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"D"}, delete: true},
			},
			mapping: map[int]int{0: 1, 1: 0, 2: 3, 3: 2}, // A->1, B->0, C->3, D->2
			expectedActions: []*action{
				{
					actionType: actionTypeDelete,
					index:      3, // 後ろから削除するので、まずDが削除される（index 3）
					slide:      &Slide{Layout: "title", Titles: []string{"D"}, delete: true},
				},
				{
					actionType: actionTypeDelete,
					index:      1, // 次にBが削除される（index 1）
					slide:      &Slide{Layout: "title", Titles: []string{"B"}, delete: true},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			expectedMapping: map[int]int{0: 1, 1: 3}, // A->1, C->3 (Cは元々index 2だったが、Bが削除されたのでindex 1になる)
		},
		{
			name: "all slides deleted",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}, delete: true},
				{Layout: "title", Titles: []string{"B"}, delete: true},
			},
			mapping: map[int]int{0: 0, 1: 1},
			expectedActions: []*action{
				{
					actionType: actionTypeDelete,
					index:      1, // 後ろから削除するので、まずBが削除される（index 1）
					slide:      &Slide{Layout: "title", Titles: []string{"B"}, delete: true},
				},
				{
					actionType: actionTypeDelete,
					index:      0, // 次にAが削除される（index 0）
					slide:      &Slide{Layout: "title", Titles: []string{"A"}, delete: true},
				},
			},
			expectedBefore:  Slides{},
			expectedMapping: map[int]int{},
		},
		{
			name: "complex delete with reordering",
			before: Slides{
				{Layout: "title", Titles: []string{"Keep1"}},                 // index 0
				{Layout: "title", Titles: []string{"Delete1"}, delete: true}, // index 1
				{Layout: "title", Titles: []string{"Keep2"}},                 // index 2
				{Layout: "title", Titles: []string{"Delete2"}, delete: true}, // index 3
				{Layout: "title", Titles: []string{"Keep3"}},                 // index 4
			},
			mapping: map[int]int{0: 2, 1: 0, 2: 4, 3: 1, 4: 3}, // Keep1->2, Delete1->0, Keep2->4, Delete2->1, Keep3->3
			expectedActions: []*action{
				{
					actionType: actionTypeDelete,
					index:      3, // 後ろから削除するので、まずDelete2が削除される（index 3）
					slide:      &Slide{Layout: "title", Titles: []string{"Delete2"}, delete: true},
				},
				{
					actionType: actionTypeDelete,
					index:      1, // 次にDelete1が削除される（index 1）
					slide:      &Slide{Layout: "title", Titles: []string{"Delete1"}, delete: true},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"Keep1"}}, // index 0
				{Layout: "title", Titles: []string{"Keep2"}}, // index 1 (元々index 2)
				{Layout: "title", Titles: []string{"Keep3"}}, // index 2 (元々index 4)
			},
			expectedMapping: map[int]int{0: 2, 1: 4, 2: 3}, // Keep1->2, Keep2->4, Keep3->3 (調整後)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// beforeとmappingのコピーを作成（元のデータを保護）
			beforeCopy := make(Slides, len(tt.before))
			for i, slide := range tt.before {
				beforeCopy[i] = copySlide(slide)
			}

			mappingCopy := make(map[int]int)
			maps.Copy(mappingCopy, tt.mapping)

			// 関数実行
			actions := generateDeleteActions(&beforeCopy, &mappingCopy)

			// アクション数の検証
			if len(actions) != len(tt.expectedActions) {
				t.Errorf("generateDeleteActions() action count = %d, expected %d", len(actions), len(tt.expectedActions))
			}

			// 各アクションの検証
			for i, action := range actions {
				if i >= len(tt.expectedActions) {
					break
				}
				expected := tt.expectedActions[i]

				if action.actionType != expected.actionType {
					t.Errorf("generateDeleteActions() action[%d].actionType = %v, expected %v", i, action.actionType, expected.actionType)
				}

				if action.index != expected.index {
					t.Errorf("generateDeleteActions() action[%d].index = %d, expected %d", i, action.index, expected.index)
				}

				// スライドの内容検証（簡易版）
				if action.slide != nil && expected.slide != nil {
					if len(action.slide.Titles) > 0 && len(expected.slide.Titles) > 0 {
						if action.slide.Titles[0] != expected.slide.Titles[0] {
							t.Errorf("generateDeleteActions() action[%d].slide.Titles[0] = %s, expected %s", i, action.slide.Titles[0], expected.slide.Titles[0])
						}
					}
				}
			}

			// beforeの状態検証
			if len(beforeCopy) != len(tt.expectedBefore) {
				t.Errorf("generateDeleteActions() before length = %d, expected %d", len(beforeCopy), len(tt.expectedBefore))
			}

			for i, slide := range beforeCopy {
				if i >= len(tt.expectedBefore) {
					break
				}
				expected := tt.expectedBefore[i]
				if len(slide.Titles) > 0 && len(expected.Titles) > 0 {
					if slide.Titles[0] != expected.Titles[0] {
						t.Errorf("generateDeleteActions() before[%d].Titles[0] = %s, expected %s", i, slide.Titles[0], expected.Titles[0])
					}
				}
			}

			// mappingの状態検証
			if len(mappingCopy) != len(tt.expectedMapping) {
				t.Errorf("generateDeleteActions() mapping length = %d, expected %d", len(mappingCopy), len(tt.expectedMapping))
			}

			for k, v := range tt.expectedMapping {
				if actualV, ok := mappingCopy[k]; !ok {
					t.Errorf("generateDeleteActions() mapping missing key %d", k)
				} else if actualV != v {
					t.Errorf("generateDeleteActions() mapping[%d] = %d, expected %d", k, actualV, v)
				}
			}
		})
	}
}

// TestGenerateMoveActions tests the generateMoveActions function.
func TestGenerateMoveActions(t *testing.T) {
	tests := []struct {
		name            string
		before          Slides
		after           Slides
		mapping         map[int]int
		expectedActions []*action
		expectedBefore  Slides // beforeの期待される状態（移動後）
	}{
		{
			name: "no moves needed - already in correct order",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			mapping:         map[int]int{0: 0, 1: 1},
			expectedActions: []*action{},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
		},
		{
			name: "simple swap - two slides",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
			mapping: map[int]int{0: 1, 1: 0}, // A->1, B->0
			// 正しい期待値: B(index 1)をindex 0に移動するだけで済む
			expectedActions: []*action{
				{
					actionType:  actionTypeMove,
					index:       1, // Bを移動
					moveToIndex: 0,
					slide:       &Slide{Layout: "title", Titles: []string{"B"}},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
		},
		{
			name: "three slides reordering - A B C to C A B",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
			mapping: map[int]int{0: 1, 1: 2, 2: 0}, // A->1, B->2, C->0
			// 正しい期待値: C(index 2)をindex 0に移動するだけで済む
			expectedActions: []*action{
				{
					actionType:  actionTypeMove,
					index:       2, // Cを移動
					moveToIndex: 0,
					slide:       &Slide{Layout: "title", Titles: []string{"C"}},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
			},
		},
		{
			name: "complex reordering - A B C D to D B A C",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"D"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"D"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			mapping: map[int]int{0: 2, 1: 1, 2: 3, 3: 0}, // A->2, B->1, C->3, D->0
			// 正しい期待値: 2つの移動が必要
			// 1. D(index 3)をindex 0に移動 → D A B C
			// 2. B(index 2)をindex 1に移動 → D B A C
			expectedActions: []*action{
				{
					actionType:  actionTypeMove,
					index:       3, // Dを移動
					moveToIndex: 0,
					slide:       &Slide{Layout: "title", Titles: []string{"D"}},
				},
				{
					actionType:  actionTypeMove,
					index:       2, // Bを移動（Dが移動した後の位置）
					moveToIndex: 1,
					slide:       &Slide{Layout: "title", Titles: []string{"B"}},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"D"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"C"}},
			},
		},
		{
			name: "single slide - no moves needed",
			before: Slides{
				{Layout: "title", Titles: []string{"Only"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Only"}},
			},
			mapping:         map[int]int{0: 0},
			expectedActions: []*action{},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"Only"}},
			},
		},
		{
			name: "reverse order - A B C to C B A",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
			mapping: map[int]int{0: 2, 1: 1, 2: 0}, // A->2, B->1, C->0
			// 正しい期待値: 2つの移動が必要
			// 1. C(index 2)をindex 0に移動 → C A B
			// 2. B(index 2)をindex 1に移動 → C B A
			expectedActions: []*action{
				{
					actionType:  actionTypeMove,
					index:       2, // Cを移動
					moveToIndex: 0,
					slide:       &Slide{Layout: "title", Titles: []string{"C"}},
				},
				{
					actionType:  actionTypeMove,
					index:       2, // Bを移動（Cが移動した後の位置）
					moveToIndex: 1,
					slide:       &Slide{Layout: "title", Titles: []string{"B"}},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"C"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
		},
		{
			name: "complex reordering - A A B A to A B A A",
			before: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"A"}},
			},
			mapping: map[int]int{0: 0, 1: 2, 2: 1, 3: 3},
			// 正しい期待値: 1つの移動が必要
			// 1. B(index 2)をindex 1に移動 →A B A A
			expectedActions: []*action{
				{
					actionType:  actionTypeMove,
					index:       2, // Bを移動
					moveToIndex: 1,
					slide:       &Slide{Layout: "title", Titles: []string{"B"}},
				},
			},
			expectedBefore: Slides{
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"B"}},
				{Layout: "title", Titles: []string{"A"}},
				{Layout: "title", Titles: []string{"A"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// beforeとmappingのコピーを作成（元のデータを保護）
			beforeCopy := make(Slides, len(tt.before))
			for i, slide := range tt.before {
				beforeCopy[i] = copySlide(slide)
			}

			mappingCopy := make(map[int]int)
			maps.Copy(mappingCopy, tt.mapping)

			// 関数実行
			actions := generateMoveActions(&beforeCopy, tt.after, &mappingCopy)

			// アクション数の検証
			if len(actions) != len(tt.expectedActions) {
				t.Errorf("generateMoveActions() action count = %d, expected %d", len(actions), len(tt.expectedActions))
				// デバッグ情報を出力
				for i, action := range actions {
					t.Logf("actual action[%d]: type=%s, index=%d, moveToIndex=%d", i, action.actionType.String(), action.index, action.moveToIndex)
				}
				for i, action := range tt.expectedActions {
					t.Logf("expected action[%d]: type=%s, index=%d, moveToIndex=%d", i, action.actionType.String(), action.index, action.moveToIndex)
				}
			}

			// 各アクションの検証
			for i, action := range actions {
				if i >= len(tt.expectedActions) {
					break
				}
				expected := tt.expectedActions[i]

				if action.actionType != expected.actionType {
					t.Errorf("generateMoveActions() action[%d].actionType = %v, expected %v", i, action.actionType, expected.actionType)
				}

				if action.index != expected.index {
					t.Errorf("generateMoveActions() action[%d].index = %d, expected %d", i, action.index, expected.index)
				}

				if action.moveToIndex != expected.moveToIndex {
					t.Errorf("generateMoveActions() action[%d].moveToIndex = %d, expected %d", i, action.moveToIndex, expected.moveToIndex)
				}

				// スライドの内容検証（簡易版）
				if action.slide != nil && expected.slide != nil {
					if len(action.slide.Titles) > 0 && len(expected.slide.Titles) > 0 {
						if action.slide.Titles[0] != expected.slide.Titles[0] {
							t.Errorf("generateMoveActions() action[%d].slide.Titles[0] = %s, expected %s", i, action.slide.Titles[0], expected.slide.Titles[0])
						}
					}
				}
			}

			// beforeの最終状態検証
			if len(beforeCopy) != len(tt.expectedBefore) {
				t.Errorf("generateMoveActions() before length = %d, expected %d", len(beforeCopy), len(tt.expectedBefore))
			}

			for i, slide := range beforeCopy {
				if i >= len(tt.expectedBefore) {
					break
				}
				expected := tt.expectedBefore[i]
				if len(slide.Titles) > 0 && len(expected.Titles) > 0 {
					if slide.Titles[0] != expected.Titles[0] {
						t.Errorf("generateMoveActions() before[%d].Titles[0] = %s, expected %s", i, slide.Titles[0], expected.Titles[0])
					}
				}
			}
		})
	}
}

func actionsEmulator(t *testing.T, before Slides, actions []*action) Slides {
	t.Helper()
	beforeCopy, err := copySlides(before)
	if err != nil {
		t.Fatalf("actionsEmulator() failed to deep copy before slides: %v", err)
	}
	for _, action := range actions {
		switch action.actionType {
		case actionTypeAppend:
			if action.slide == nil {
				t.Fatalf("actionsEmulator() append action missing slide")
			}
			beforeCopy = append(beforeCopy, copySlide(action.slide))
		case actionTypeUpdate:
			if action.slide == nil {
				t.Fatalf("actionsEmulator() update action missing slide")
			}
			if action.index < 0 || action.index >= len(beforeCopy) {
				t.Fatalf("actionsEmulator() update action invalid index: %d", action.index)
			}
			beforeCopy[action.index] = copySlide(action.slide)
		case actionTypeDelete:
			if action.index < 0 || action.index >= len(beforeCopy) {
				t.Fatalf("actionsEmulator() delete action invalid index: %d", action.index)
			}
			beforeCopy = slices.Delete(beforeCopy, action.index, action.index+1)
		case actionTypeMove:
			if action.index < 0 || action.index >= len(beforeCopy) {
				t.Fatalf("actionsEmulator() move action invalid index: %d", action.index)
			}
			if action.moveToIndex < 0 || action.moveToIndex >= len(beforeCopy) {
				t.Fatalf("actionsEmulator() move action invalid moveToIndex: %d", action.moveToIndex)
			}
			slide := copySlide(beforeCopy[action.index])
			beforeCopy = append(beforeCopy[:action.moveToIndex], append([]*Slide{slide}, beforeCopy[action.moveToIndex:]...)...)
			var deleteIndex int
			if action.index < action.moveToIndex {
				deleteIndex = action.index
			} else {
				deleteIndex = action.index + 1
			}
			beforeCopy = slices.Delete(beforeCopy, deleteIndex, deleteIndex+1)
		default:
			t.Fatalf("actionsEmulator() unknown action type: %v", action.actionType)
		}
	}
	return beforeCopy
}
