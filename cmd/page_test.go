package cmd

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPageToPages(t *testing.T) {
	tests := []struct {
		page    string
		total   int
		want    []int
		wantErr bool
	}{
		{"3", 10, []int{3}, false},
		{"1,3,4", 10, []int{1, 3, 4}, false},
		{"3-", 10, []int{3, 4, 5, 6, 7, 8, 9, 10}, false},
		{"-5", 10, []int{1, 2, 3, 4, 5}, false},
		{"3-5", 10, []int{3, 4, 5}, false},
	}
	for _, tt := range tests {
		t.Run(tt.page, func(t *testing.T) {
			got, err := pageToPages(tt.page, tt.total)
			if (err != nil) != tt.wantErr {
				t.Errorf("pageToPages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("pageToPages() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
