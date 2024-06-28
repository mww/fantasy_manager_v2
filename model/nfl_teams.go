package model

import (
	"fmt"
	"strings"
)

type NFLTeam struct {
	name   string
	loc    string
	mascot string
	short  string   // If there is short form of the name, e.g. SF for SFO
	nick   []string // Any other nicknames that are used for the team, e.g. Philly for PHI
}

func (t *NFLTeam) String() string {
	return t.name
}

func (t *NFLTeam) Friendly() string {
	if t.loc == "" {
		return t.name
	}
	return fmt.Sprintf("%s %s", t.loc, t.mascot)
}

func (t *NFLTeam) Equals(o *NFLTeam) bool {
	if o == nil {
		return false
	}

	if t == o {
		return true
	}

	return t.name == o.name &&
		t.loc == o.loc &&
		t.mascot == o.mascot &&
		t.short == o.short &&
		arrayEquals(t.nick, o.nick)
}

var (
	TEAM_FA *NFLTeam = &NFLTeam{name: "FA", nick: []string{"FA*"}}

	// NFC
	TEAM_ARI *NFLTeam = &NFLTeam{name: "ARI", loc: "Arizona", mascot: "Cardinals", nick: []string{"Cards"}}
	TEAM_ATL *NFLTeam = &NFLTeam{name: "ATL", loc: "Atlanta", mascot: "Falcons"}
	TEAM_CAR *NFLTeam = &NFLTeam{name: "CAR", loc: "Carolina", mascot: "Panthers"}
	TEAM_CHI *NFLTeam = &NFLTeam{name: "CHI", loc: "Chicago", mascot: "Bears"}
	TEAM_DAL *NFLTeam = &NFLTeam{name: "DAL", loc: "Dallas", mascot: "Cowboys"}
	TEAM_DET *NFLTeam = &NFLTeam{name: "DET", loc: "Detroit", mascot: "Lions"}
	TEAM_GBP *NFLTeam = &NFLTeam{name: "GBP", loc: "Green Bay", mascot: "Packers", short: "GB"}
	TEAM_LAR *NFLTeam = &NFLTeam{name: "LAR", loc: "Los Angeles", mascot: "Rams"}
	TEAM_MIN *NFLTeam = &NFLTeam{name: "MIN", loc: "Minnesota", mascot: "Vikings"}
	TEAM_NOS *NFLTeam = &NFLTeam{name: "NOS", loc: "New Orleans", mascot: "Saints", short: "NO"}
	TEAM_NYG *NFLTeam = &NFLTeam{name: "NYG", loc: "New York", mascot: "Giants"}
	TEAM_PHI *NFLTeam = &NFLTeam{name: "PHI", loc: "Philadelphia", mascot: "Eagles", nick: []string{"Philly"}}
	TEAM_SFO *NFLTeam = &NFLTeam{name: "SFO", loc: "San Francisco", mascot: "49ers", short: "SF", nick: []string{"Niners", "9ers"}}
	TEAM_SEA *NFLTeam = &NFLTeam{name: "SEA", loc: "Seattle", mascot: "Seahawks", nick: []string{"Hawks"}}
	TEAM_TBB *NFLTeam = &NFLTeam{name: "TBB", loc: "Tampa Bay", mascot: "Buccaneers", short: "TB", nick: []string{"Bucks"}}
	TEAM_WAS *NFLTeam = &NFLTeam{name: "WAS", loc: "Washington", mascot: "Commanders"}

	// AFC
	TEAM_BAL *NFLTeam = &NFLTeam{name: "BAL", loc: "Baltimore", mascot: "Ravens"}
	TEAM_BUF *NFLTeam = &NFLTeam{name: "BUF", loc: "Buffalo", mascot: "Bills"}
	TEAM_CIN *NFLTeam = &NFLTeam{name: "CIN", loc: "Cincinnati", mascot: "Bangals"}
	TEAM_CLE *NFLTeam = &NFLTeam{name: "CLE", loc: "Cleveland", mascot: "Browns"}
	TEAM_DEN *NFLTeam = &NFLTeam{name: "DEN", loc: "Denver", mascot: "Broncos"}
	TEAM_HOU *NFLTeam = &NFLTeam{name: "HOU", loc: "Houston", mascot: "Texans"}
	TEAM_IND *NFLTeam = &NFLTeam{name: "IND", loc: "Indianapolis", mascot: "Colts", nick: []string{"Indy"}}
	TEAM_JAC *NFLTeam = &NFLTeam{name: "JAC", loc: "Jacksonville", mascot: "Jaguars", nick: []string{"Jags"}}
	TEAM_KCC *NFLTeam = &NFLTeam{name: "KCC", loc: "Kansas City", mascot: "Chiefs", short: "KC"}
	TEAM_LVR *NFLTeam = &NFLTeam{name: "LVR", loc: "Las Vegas", mascot: "Raiders", short: "LV"}
	TEAM_LAC *NFLTeam = &NFLTeam{name: "LAC", loc: "Los Angeles", mascot: "Chargers"}
	TEAM_MIA *NFLTeam = &NFLTeam{name: "MIA", loc: "Miami", mascot: "Dolphins"}
	TEAM_NEP *NFLTeam = &NFLTeam{name: "NEP", loc: "New England", mascot: "Patriots", short: "NE", nick: []string{"Pats"}}
	TEAM_NYJ *NFLTeam = &NFLTeam{name: "NYJ", loc: "New York", mascot: "Jets"}
	TEAM_PIT *NFLTeam = &NFLTeam{name: "PIT", loc: "Pittsburgh", mascot: "Steelers", nick: []string{"Pitt"}}
	TEAM_TEN *NFLTeam = &NFLTeam{name: "TEN", loc: "Tennessee", mascot: "Titans"}

	teamMap map[string]*NFLTeam = buildTeamMap()
)

func ParseTeam(name string) *NFLTeam {
	t := teamMap[strings.ToLower(name)]
	if t == nil {
		return TEAM_FA
	}
	return t
}

func buildTeamMap() map[string]*NFLTeam {
	teams := []*NFLTeam{
		// NFC
		TEAM_ARI, TEAM_ATL, TEAM_CAR, TEAM_CHI, TEAM_DAL, TEAM_DET, TEAM_GBP, TEAM_LAR,
		TEAM_MIN, TEAM_NOS, TEAM_NYG, TEAM_PHI, TEAM_SFO, TEAM_SEA, TEAM_TBB, TEAM_WAS,
		// AFC
		TEAM_BAL, TEAM_BUF, TEAM_CIN, TEAM_CLE, TEAM_DEN, TEAM_HOU, TEAM_IND, TEAM_JAC,
		TEAM_KCC, TEAM_LVR, TEAM_LAC, TEAM_MIA, TEAM_NEP, TEAM_NYJ, TEAM_PIT, TEAM_TEN,
		// Other
		TEAM_FA,
	}

	teamMap := make(map[string]*NFLTeam)
	for _, t := range teams {
		teamMap[strings.ToLower(t.name)] = t

		if t.loc != "" {
			teamMap[strings.ToLower(t.loc)] = t
		}

		if t.mascot != "" {
			teamMap[strings.ToLower(t.mascot)] = t
		}

		if t.short != "" {
			teamMap[strings.ToLower(t.short)] = t
		}

		for _, n := range t.nick {
			teamMap[strings.ToLower(n)] = t
		}
	}
	return teamMap
}

func arrayEquals(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}

	return true
}
