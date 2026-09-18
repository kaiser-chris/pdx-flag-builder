package gui

import (
	"reflect"
	"testing"
)

func TestMatchRanges(t *testing.T) {
	tests := []struct {
		text, query string
		want        [][2]int
	}{
		{"ce_bear_california.dds", "bear", [][2]int{{3, 7}}},
		{"ce_bear_california.dds", "BEAR", [][2]int{{3, 7}}},
		{"ce_bear_california.dds", "  bear ", [][2]int{{3, 7}}},
		{"pattern_border_of_2_double.dds", "o", [][2]int{{9, 10}, {15, 16}, {21, 22}}},
		{"aaaa", "aa", [][2]int{{0, 2}, {2, 4}}},
		{"GBR", "", nil},
		{"GBR", "   ", nil},
		{"GBR", "FRA", nil},
	}

	for _, test := range tests {
		if got := MatchRanges(test.text, test.query); !reflect.DeepEqual(got, test.want) {
			t.Errorf("MatchRanges(%q, %q) = %v, want %v", test.text, test.query, got, test.want)
		}
	}
}
