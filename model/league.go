package model

var PlatformSleeper = "sleeper"

type League struct {
	ID         int32
	Platform   string
	ExternalID string
	Name       string
	Year       string
	Archived   bool
}

func IsPlatformSupported(platform string) bool {
	return platform == PlatformSleeper
}
