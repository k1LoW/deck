package deck

import (
	"testing"
)

func TestSimilarityCalculation(t *testing.T) {
	tests := []struct {
		name     string
		slide1   *Slide
		slide2   *Slide
		expected int
	}{
		{
			name: "identical slides",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			expected: 500, // perfect match
		},
		{
			name: "same layout and title",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Same Title"},
			},
			slide2: &Slide{
				Layout:    "title",
				Titles:    []string{"Same Title"},
				Subtitles: []string{"Different Subtitle"},
			},
			expected: 90, // layout (10) + title (80)
		},
		{
			name: "same layout only",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title 2"},
			},
			expected: 10, // layout only
		},
		{
			name: "no similarity",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title 1"},
			},
			slide2: &Slide{
				Layout: "section",
				Titles: []string{"Title 2"},
			},
			expected: 0, // no match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getSimilarity(tt.slide1, tt.slide2)
			if actual != tt.expected {
				t.Errorf("Expected similarity %d, got %d", tt.expected, actual)
			}
		})
	}
}

func TestSimilarityForMapping(t *testing.T) {
	slide1 := &Slide{Layout: "title", Titles: []string{"Title"}}
	slide2 := &Slide{Layout: "title", Titles: []string{"Title"}}

	tests := []struct {
		name          string
		beforeIndex   int
		afterIndex    int
		expectedBonus int
	}{
		{
			name:          "same position",
			beforeIndex:   0,
			afterIndex:    0,
			expectedBonus: 4, // perfect position match
		},
		{
			name:          "forward movement",
			beforeIndex:   0,
			afterIndex:    1,
			expectedBonus: 2, // natural order
		},
		{
			name:          "backward movement",
			beforeIndex:   1,
			afterIndex:    0,
			expectedBonus: 0, // no bonus
		},
	}

	baseSimilarity := getSimilarity(slide1, slide2)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := getSimilarityForMapping(slide1, slide2, tt.beforeIndex, tt.afterIndex)
			expected := baseSimilarity + tt.expectedBonus
			if actual != expected {
				t.Errorf("Expected similarity %d (base %d + bonus %d), got %d",
					expected, baseSimilarity, tt.expectedBonus, actual)
			}
		})
	}
}
