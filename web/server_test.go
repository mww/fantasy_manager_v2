package web

import (
	"testing"
	"time"
)

func TestAgeFormatterInternal(t *testing.T) {
	end := getDate(2024, 6, 28)
	tests := []struct {
		birth time.Time
		want  string
	}{
		{birth: getDate(2000, 1, 1), want: "Jan 1, 2000 (24 years, 179 days)"},
		{birth: getDate(2001, 7, 4), want: "Jul 4, 2001 (22 years, 360 days)"},
		{birth: getDate(2021, 2, 5), want: "Feb 5, 2021 (3 years, 144 days)"},
		{birth: getDate(1982, 9, 25), want: "Sep 25, 1982 (41 years, 277 days)"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := ageFormatterInternal(tc.birth, end)
			if tc.want != got {
				t.Errorf("expected: '%v', got: '%v", tc.want, got)
			}
		})
	}
}

func TestHeightFormatter(t *testing.T) {
	tests := []struct {
		h    int
		want string
	}{
		{h: 12, want: "1'0\""},
		{h: 24, want: "2'0\""},
		{h: 36, want: "3'0\""},
		{h: 48, want: "4'0\""},
		{h: 60, want: "5'0\""},
		{h: 72, want: "6'0\""},
		{h: 84, want: "7'0\""},
		{h: 10, want: "0'10\""},
		{h: 70, want: "5'10\""},
		{h: 73, want: "6'1\""},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := heightFormatter(tc.h)
			if tc.want != got {
				t.Errorf("expected: '%v', got: '%v'", tc.want, got)
			}
		})
	}
}

func TestDateFormatter(t *testing.T) {
	tests := []struct {
		d    time.Time
		want string
	}{
		{d: getDate(2021, 8, 23), want: "2021-08-23"},
		{d: getDate(2019, 9, 3), want: "2019-09-03"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := dateFormatter(tc.d)
			if tc.want != got {
				t.Errorf("expected: '%v', got: '%v'", tc.want, got)
			}
		})
	}
}

func TestYearFormatter(t *testing.T) {
	tests := []struct {
		d    time.Time
		want string
	}{
		{d: getDate(2021, 8, 23), want: "2021"},
		{d: getDate(2019, 9, 3), want: "2019"},
		{d: getDate(1982, 9, 25), want: "1982"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := yearFormatter(tc.d)
			if tc.want != got {
				t.Errorf("expected: '%v', got: '%v'", tc.want, got)
			}
		})
	}
}

func getDate(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}
