package deck

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveFrozenSlides(t *testing.T) {
	tests := []struct {
		name   string
		before Slides
		after  Slides
		want   map[int]int
	}{
		{
			name:   "empty slides",
			before: Slides{},
			after:  Slides{},
			want:   map[int]int{},
		},
		{
			name: "no frozen slides",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
			},
			want: map[int]int{},
		},
		{
			name:   "frozen slide with empty before",
			before: Slides{},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}, Freeze: true},
			},
			want: map[int]int{},
		},
		{
			name: "insert before frozen slide",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide 2"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide NEW"}},
				{Layout: "title", Titles: []string{"Slide 2"}, Freeze: true},
			},
			want: map[int]int{2: 1},
		},
		{
			name: "delete before frozen slide",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide 2"}},
				{Layout: "title", Titles: []string{"Slide 3"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide 3"}, Freeze: true},
			},
			want: map[int]int{1: 2},
		},
		{
			// The frozen slide's actual content has diverged from its
			// markdown source. It must still adopt a counterpart by
			// relative order between anchors.
			name: "diverged frozen slide adopts counterpart by order",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide 2 edited manually"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide 2"}, Freeze: true},
			},
			want: map[int]int{1: 1},
		},
		{
			// The frozen slide crosses the anchor "Slide C" in the desired
			// order. The order-preserving alignment cannot express this
			// move, so the second pass matches it by similarity.
			name: "frozen slide moved across anchors",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide A"}},
				{Layout: "title", Titles: []string{"Slide B"}},
				{Layout: "title", Titles: []string{"Slide C"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide A"}},
				{Layout: "title", Titles: []string{"Slide C"}},
				{Layout: "title", Titles: []string{"Slide B"}, Freeze: true},
			},
			want: map[int]int{2: 1},
		},
		{
			// Every before slide is claimed by a non-frozen anchor, so the
			// frozen slide has no counterpart and is absent from the result.
			name: "frozen slide without counterpart",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide 1"}},
				{Layout: "title", Titles: []string{"Slide F"}, Freeze: true},
			},
			want: map[int]int{},
		},
		{
			name: "all slides frozen with insert between",
			before: Slides{
				{Layout: "title", Titles: []string{"Slide P"}},
				{Layout: "title", Titles: []string{"Slide Q"}},
			},
			after: Slides{
				{Layout: "title", Titles: []string{"Slide P"}, Freeze: true},
				{Layout: "title", Titles: []string{"Slide NEW"}},
				{Layout: "title", Titles: []string{"Slide Q"}, Freeze: true},
			},
			want: map[int]int{0: 0, 2: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveFrozenSlides(tt.before, tt.after)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}
