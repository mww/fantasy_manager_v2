package model

import (
	"testing"
	"time"
)

func TestPlayerDateFormatFunctions(t *testing.T) {
	zeroDates := &Player{
		BirthDate:  time.Time{},
		RookieYear: time.Time{},
		Created:    time.Time{},
		Updated:    time.Time{},
	}
	if zeroDates.FormattedBirthDate() != "unknown" {
		t.Error("birthdate is not unknown")
	}
	if zeroDates.FormattedRookieYear() != "unknown" {
		t.Error("rookie year is not unknown")
	}
	if zeroDates.FormattedCreatedTime() != "unknown" {
		t.Error("created time is not unknown")
	}
	if zeroDates.FormattedUpdatedTime() != "unknown" {
		t.Error("updated time is not unknown")
	}

	nonZeroDates := &Player{
		BirthDate:  time.Date(1994, 2, 20, 0, 0, 0, 0, time.UTC),
		RookieYear: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
		Created:    time.Date(2024, 7, 8, 10, 14, 36, 136, time.UTC),
		Updated:    time.Date(2024, 7, 9, 17, 12, 51, 0, time.UTC),
	}
	if nonZeroDates.FormattedBirthDate() != "1994-02-20" {
		t.Errorf("birthdate was not expected value: '%s'", nonZeroDates.FormattedBirthDate())
	}
	if nonZeroDates.FormattedRookieYear() != "2015" {
		t.Errorf("rookie year was not expected value: '%s'", nonZeroDates.FormattedRookieYear())
	}
	if nonZeroDates.FormattedCreatedTime() != "2024-07-08 10:14:36" {
		t.Errorf("created time was not expected value: '%s'", nonZeroDates.FormattedCreatedTime())
	}
	if nonZeroDates.FormattedUpdatedTime() != "2024-07-09 17:12:51" {
		t.Errorf("updated time was not expected value: '%s'", nonZeroDates.FormattedUpdatedTime())
	}
}

func TestChangeString(t *testing.T) {
	c := &Change{
		Time:         time.Date(2024, 7, 8, 10, 23, 19, 111, time.UTC),
		PropertyName: "FirstName",
		OldValue:     "Jonny",
		NewValue:     "John",
	}
	if c.String() != "FirstName changed from 'Jonny' to 'John'" {
		t.Errorf("string was not expected value: '%s'", c.String())
	}
}
