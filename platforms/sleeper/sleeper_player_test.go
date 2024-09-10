package sleeper

import (
	"testing"
	"time"
)

func TestParseHeight(t *testing.T) {
	tests := map[string]struct {
		input string
		want  int
	}{
		"empty input":                {input: "", want: 0},
		"height in inches":           {input: "75", want: 75},
		"height in feet and inches":  {input: `6'2"`, want: 74},
		"letters in input":           {input: "bad", want: 0},
		"number letter mix":          {input: "72bad", want: 0},
		"numbers in feet and inches": {input: `5'abc"`, want: 0},
		"five feet zero inches":      {input: `5'0"`, want: 60},
		"zero feet seven inches":     {input: `0'7"`, want: 0},
		"five feet 03 inches":        {input: `5'03"`, want: 63},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseHeight(tc.input, "123")
			if got != tc.want {
				t.Fatalf("expected: %d, got: %d", tc.want, got)
			}
		})
	}
}

func TestParseBirthday(t *testing.T) {
	tests := map[string]struct {
		input string
		want  time.Time
	}{
		"empty input":     {input: "", want: zeroTime},
		"normal birthday": {input: "1990-12-09", want: time.Date(1990, time.December, 9, 0, 0, 0, 0, time.UTC)},
		"birthday two":    {input: "2002-02-14", want: time.Date(2002, time.February, 14, 0, 0, 0, 0, time.UTC)},
		"wrong format":    {input: "2002/2/14", want: zeroTime},
		"US format":       {input: "02-14-2002", want: zeroTime},
		"missing zero":    {input: "2002-2-14", want: zeroTime},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseBirthdate(tc.input, "123")
			if got != tc.want {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestParseRookieYear(t *testing.T) {
	tests := map[string]struct {
		input *metadata
		want  time.Time
	}{
		"nil":          {input: nil, want: zeroTime},
		"2023":         {input: &metadata{RookieYear: "2023"}, want: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		"empty":        {input: &metadata{RookieYear: ""}, want: zeroTime},
		"bad date":     {input: &metadata{RookieYear: "nine"}, want: zeroTime},
		"too specific": {input: &metadata{RookieYear: "2023-05-17"}, want: zeroTime},
		"zero":         {input: &metadata{RookieYear: "0"}, want: zeroTime},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := parseRookieYear(tc.input, "123")
			if got != tc.want {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}
