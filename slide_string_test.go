package deck

import (
	"testing"
)

func TestBodyString(t *testing.T) {
	tests := []struct {
		name     string
		body     *Body
		expected string
	}{
		{
			name:     "empty body",
			body:     &Body{},
			expected: "",
		},
		{
			name: "single paragraph without bullet",
			body: &Body{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Hello world"},
						},
						Bullet: BulletNone,
					},
				},
			},
			expected: "Hello world\n",
		},
		{
			name: "single paragraph with dash bullet",
			body: &Body{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "First item"},
						},
						Bullet: BulletDash,
					},
				},
			},
			expected: "- First item\n",
		},
		{
			name: "multiple paragraphs mixed bullets and text",
			body: &Body{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Introduction text"},
						},
						Bullet: BulletNone,
					},
					{
						Fragments: []*Fragment{
							{Value: "First bullet point"},
						},
						Bullet: BulletDash,
					},
					{
						Fragments: []*Fragment{
							{Value: "Second bullet point"},
						},
						Bullet: BulletDash,
					},
				},
			},
			expected: "Introduction text\n\n- First bullet point\n- Second bullet point\n",
		},
		{
			name: "multiple fragments in single paragraph",
			body: &Body{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Hello "},
							{Value: "beautiful "},
							{Value: "world"},
						},
						Bullet: BulletNone,
					},
				},
			},
			expected: "Hello beautiful world\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.body.String()
			if got != tt.expected {
				t.Errorf("Body.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestParagraphString(t *testing.T) {
	tests := []struct {
		name      string
		paragraph *Paragraph
		expected  string
	}{
		{
			name:      "nil paragraph",
			paragraph: nil,
			expected:  "",
		},
		{
			name: "paragraph with no bullet and no nesting",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Simple text"},
				},
				Bullet:  BulletNone,
				Nesting: 0,
			},
			expected: "Simple text",
		},
		{
			name: "paragraph with dash bullet",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Dash bullet item"},
				},
				Bullet:  BulletDash,
				Nesting: 0,
			},
			expected: "- Dash bullet item",
		},
		{
			name: "paragraph with number bullet",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Number bullet item"},
				},
				Bullet:  BulletNumber,
				Nesting: 0,
			},
			expected: "1. Number bullet item",
		},
		{
			name: "paragraph with alpha bullet",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Alpha bullet item"},
				},
				Bullet:  BulletAlpha,
				Nesting: 0,
			},
			expected: "a. Alpha bullet item",
		},
		{
			name: "paragraph with nesting level 1",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Nested item"},
				},
				Bullet:  BulletDash,
				Nesting: 1,
			},
			expected: "  - Nested item",
		},
		{
			name: "paragraph with nesting level 2",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Double nested item"},
				},
				Bullet:  BulletDash,
				Nesting: 2,
			},
			expected: "    - Double nested item",
		},
		{
			name: "paragraph with multiple fragments",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "First "},
					{Value: "second "},
					{Value: "third"},
				},
				Bullet:  BulletNone,
				Nesting: 0,
			},
			expected: "First second third",
		},
		{
			name: "paragraph with nil fragment",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Before nil"},
					nil,
					{Value: "After nil"},
				},
				Bullet:  BulletNone,
				Nesting: 0,
			},
			expected: "Before nilAfter nil",
		},
		{
			name: "paragraph with nested number bullet",
			paragraph: &Paragraph{
				Fragments: []*Fragment{
					{Value: "Nested numbered item"},
				},
				Bullet:  BulletNumber,
				Nesting: 1,
			},
			expected: "  1. Nested numbered item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.paragraph.String()
			if got != tt.expected {
				t.Errorf("Paragraph.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestBlockQuoteString(t *testing.T) {
	tests := []struct {
		name       string
		blockQuote *BlockQuote
		expected   string
	}{
		{
			name:       "nil blockquote",
			blockQuote: nil,
			expected:   "",
		},
		{
			name:       "empty blockquote",
			blockQuote: &BlockQuote{},
			expected:   "",
		},
		{
			name: "single paragraph without bullet",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "This is a quote"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 0,
			},
			expected: "> This is a quote\n",
		},
		{
			name: "single paragraph with dash bullet",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Quote with bullet"},
						},
						Bullet: BulletDash,
					},
				},
				Nesting: 0,
			},
			expected: "> - Quote with bullet\n",
		},
		{
			name: "multiple paragraphs without bullets",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "First paragraph"},
						},
						Bullet: BulletNone,
					},
					{
						Fragments: []*Fragment{
							{Value: "Second paragraph"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 0,
			},
			expected: "> First paragraph\n> \n> Second paragraph\n",
		},
		{
			name: "multiple paragraphs with bullets",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "First bullet"},
						},
						Bullet: BulletDash,
					},
					{
						Fragments: []*Fragment{
							{Value: "Second bullet"},
						},
						Bullet: BulletDash,
					},
				},
				Nesting: 0,
			},
			expected: "> - First bullet\n> - Second bullet\n",
		},
		{
			name: "mixed bullet and non-bullet paragraphs",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "First bullet"},
						},
						Bullet: BulletDash,
					},
					{
						Fragments: []*Fragment{
							{Value: "Plain text"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 0,
			},
			expected: "> - First bullet\n> \n> Plain text\n",
		},
		{
			name: "nested blockquote (nesting level 1)",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Nested quote"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 1,
			},
			expected: "> > Nested quote\n",
		},
		{
			name: "deeply nested blockquote (nesting level 2)",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Deeply nested quote"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 2,
			},
			expected: "> > > Deeply nested quote\n",
		},
		{
			name: "paragraph with multiple fragments",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Hello "},
							{Value: "beautiful "},
							{Value: "world"},
						},
						Bullet: BulletNone,
					},
				},
				Nesting: 0,
			},
			expected: "> Hello beautiful world\n",
		},
		{
			name: "multiple bullet types",
			blockQuote: &BlockQuote{
				Paragraphs: []*Paragraph{
					{
						Fragments: []*Fragment{
							{Value: "Dash bullet"},
						},
						Bullet: BulletDash,
					},
					{
						Fragments: []*Fragment{
							{Value: "Number bullet"},
						},
						Bullet: BulletNumber,
					},
					{
						Fragments: []*Fragment{
							{Value: "Alpha bullet"},
						},
						Bullet: BulletAlpha,
					},
				},
				Nesting: 0,
			},
			expected: "> - Dash bullet\n> 1. Number bullet\n> a. Alpha bullet\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.blockQuote.String()
			if got != tt.expected {
				t.Errorf("BlockQuote.String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
