package model

var PlatformSleeper = "sleeper"

type League struct {
	ID         int32
	Platform   string
	ExternalID string
	Name       string
	Year       string
	Archived   bool
	Managers   []LeagueManager
}

type LeagueManager struct {
	ExternalID  string
	TeamName    string
	ManagerName string
	JoinKey     string
}

func IsPlatformSupported(platform string) bool {
	return platform == PlatformSleeper
}
