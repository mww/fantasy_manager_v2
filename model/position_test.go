package model

import "testing"

func TestParsePosition(t *testing.T) {
	tests := []struct {
		input    string
		expected Position
	}{
		{input: "QB", expected: POS_QB},
		{input: "qb", expected: POS_QB},
		{input: "WR", expected: POS_WR},
		{input: "wr", expected: POS_WR},
		{input: "RB", expected: POS_RB},
		{input: "rb", expected: POS_RB},
		{input: "TE", expected: POS_TE},
		{input: "te", expected: POS_TE},
		{input: "UNKNOWN", expected: POS_UNKNOWN},
		{input: "DEF", expected: POS_UNKNOWN},
		{input: "FB", expected: POS_RB},
	}

	for _, tc := range tests {
		a := ParsePosition(tc.input)
		if a != tc.expected {
			t.Errorf("input: '%s', expected: '%s', got '%s'", tc.input, tc.expected, a)
		}
	}
}
