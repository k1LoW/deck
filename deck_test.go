package deck

import "testing"

func TestCountString(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"👉", 2},
		{"➡️", 2},
		{"👍🏼", 4},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := countString(tt.in)
			if got != tt.want {
				t.Errorf("countString(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
