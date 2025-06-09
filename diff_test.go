package deck

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

const presentationID = "1_QRwonGFKTcsakL0QFCUNvNKWMedDS-C5KRMqMTwz6E"

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

func TestApply(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		// t.Skip("skipping integration test, set TEST_INTEGRATION=1 to run")
	}

	ctx := context.Background()
	//presentationID := os.Getenv("TEST_PRESENTATION_ID")
	cmpopts := cmp.Options{
		cmpopts.IgnoreFields(Fragment{}, "ClassName", "SoftLineBreak"),
		cmpopts.IgnoreUnexported(Slide{}),
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
			before, err := d.DumpSlides(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tt.before, before, cmpopts...); diff != "" {
				t.Errorf("diff before apply: %s", diff)
			}
			t.Log("---")
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
