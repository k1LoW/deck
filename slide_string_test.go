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
