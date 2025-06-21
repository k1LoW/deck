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
			expected: 130, // layout (50) + title (80)
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
			expected: 50, // layout only
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
		{
			name: "same layout and images",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title with Image"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title with Image"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			expected: 500, // perfect match (identical slides)
		},
		{
			name: "same layout, title and different images",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title with Image"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title with Image"},
				Images: []*Image{newImage(t, "testdata/test.jpeg")},
			},
			expected: 130, // layout (50) + title (80), no image match
		},
		{
			name: "same layout and multiple images",
			slide1: &Slide{
				Layout: "content",
				Bodies: []*Body{{
					Paragraphs: []*Paragraph{{
						Fragments: []*Fragment{{Value: "Content with images"}},
					}},
				}},
				Images: []*Image{newImage(t, "testdata/test.png"), newImage(t, "testdata/test.gif")},
			},
			slide2: &Slide{
				Layout: "content",
				Bodies: []*Body{{
					Paragraphs: []*Paragraph{{
						Fragments: []*Fragment{{Value: "Content with images"}},
					}},
				}},
				Images: []*Image{newImage(t, "testdata/test.png"), newImage(t, "testdata/test.gif")},
			},
			expected: 500, // perfect match (identical slides)
		},
		{
			name: "images only in one slide",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Title"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Title"},
			},
			expected: 130, // layout (50) + title (80), no image comparison
		},
		{
			name: "different image formats same content",
			slide1: &Slide{
				Layout: "title",
				Titles: []string{"Logo Slide"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			slide2: &Slide{
				Layout: "title",
				Titles: []string{"Logo Slide"},
				Images: []*Image{newImage(t, "testdata/test.jpeg")},
			},
			expected: 130, // layout (50) + title (80), different image formats don't match
		},
		{
			name: "same layout, different images",
			slide1: &Slide{
				Layout: "content",
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			slide2: &Slide{
				Layout: "content",
				Images: []*Image{newImage(t, "img/layout.png")},
			},
			expected: 50, // layout (50) only, images don't match
		},
		{
			name: "mixed image count",
			slide1: &Slide{
				Layout: "content",
				Titles: []string{"Multiple Images"},
				Images: []*Image{newImage(t, "testdata/test.png"), newImage(t, "testdata/test.jpeg")},
			},
			slide2: &Slide{
				Layout: "content",
				Titles: []string{"Multiple Images"},
				Images: []*Image{newImage(t, "testdata/test.png")},
			},
			expected: 130, // layout (50) + title (80), different image arrays don't match
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
			expectedBonus: 8, // perfect position match for same layout
		},
		{
			name:          "forward movement",
			beforeIndex:   0,
			afterIndex:    1,
			expectedBonus: 4, // natural order for same layout
		},
		{
			name:          "backward movement",
			beforeIndex:   1,
			afterIndex:    0,
			expectedBonus: 6, // prefer earlier positions in after for same layout
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

func TestSimilarityForMappingWithImages(t *testing.T) {
	// Test mapping similarity for slides with images
	slide1 := &Slide{
		Layout: "content",
		Titles: []string{"Image Slide"},
		Images: []*Image{newImage(t, "testdata/test.png")},
	}
	slide2 := &Slide{
		Layout: "content",
		Titles: []string{"Image Slide"},
		Images: []*Image{newImage(t, "testdata/test.png")},
	}

	tests := []struct {
		name          string
		beforeIndex   int
		afterIndex    int
		expectedBonus int
	}{
		{
			name:          "same position with images",
			beforeIndex:   0,
			afterIndex:    0,
			expectedBonus: 8, // perfect position match for same layout
		},
		{
			name:          "forward movement with images",
			beforeIndex:   0,
			afterIndex:    1,
			expectedBonus: 4, // natural order for same layout
		},
		{
			name:          "backward movement with images",
			beforeIndex:   1,
			afterIndex:    0,
			expectedBonus: 6, // prefer earlier positions in after for same layout
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

func newImage(t *testing.T, pathOrURL string) *Image {
	if pathOrURL == "" {
		t.Fatal("pathOrURL cannot be empty")
	}
	img, err := NewImage(pathOrURL)
	if err != nil {
		t.Fatalf("Failed to create image from %s: %v", pathOrURL, err)
	}
	return img
}
