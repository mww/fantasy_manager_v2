package internal

type FantasyContent struct {
	League *League `xml:"league"`
	Team   *Team   `xml:"team"`
}

type League struct {
	Key       string     `xml:"league_key"`
	Name      string     `xml:"name"`
	Settings  *Settings  `xml:"settings"`
	Standings *Standings `xml:"standings"`
}

type Settings struct {
	RosterPositions *RosterPositions `xml:"roster_positions"`
}

type RosterPositions struct {
	Positions []RosterPosition `xml:"roster_position"`
}

type RosterPosition struct {
	Position string `xml:"position"`
	Count    int    `xml:"count"`
}

type Standings struct {
	Teams *Teams `xml:"teams"`
}

type Teams struct {
	Teams []Team `xml:"team"`
}

type Team struct {
	Key      string    `xml:"team_key"`
	Name     string    `xml:"name"`
	Managers *Managers `xml:"managers"`
	// Roster   *Roster   `xml:"roster"`
	// Matchups *Matchups `xml:"matchups"`
}

type Managers struct {
	Managers []Manager `xml:"manager"`
}

type Manager struct {
	Nickname string `xml:"nickname"`
}
