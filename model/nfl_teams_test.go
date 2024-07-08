package model

import "testing"

func TestParseTeam(t *testing.T) {
	tests := []struct {
		input    string
		expected *NFLTeam
	}{
		{input: "FA", expected: TEAM_FA},
		{input: "FA*", expected: TEAM_FA},

		// NFC
		{input: "ARI", expected: TEAM_ARI},
		{input: "ATL", expected: TEAM_ATL},
		{input: "CAR", expected: TEAM_CAR},
		{input: "CHI", expected: TEAM_CHI},
		{input: "DAL", expected: TEAM_DAL},
		{input: "DET", expected: TEAM_DET},
		{input: "GBP", expected: TEAM_GBP},
		{input: "LAR", expected: TEAM_LAR},
		{input: "MIN", expected: TEAM_MIN},
		{input: "NOS", expected: TEAM_NOS},
		{input: "NYG", expected: TEAM_NYG},
		{input: "PHI", expected: TEAM_PHI},
		{input: "SFO", expected: TEAM_SFO},
		{input: "SEA", expected: TEAM_SEA},
		{input: "TBB", expected: TEAM_TBB},
		{input: "WAS", expected: TEAM_WAS},

		// AFC
		{input: "BAL", expected: TEAM_BAL},
		{input: "BUF", expected: TEAM_BUF},
		{input: "CIN", expected: TEAM_CIN},
		{input: "CLE", expected: TEAM_CLE},
		{input: "DEN", expected: TEAM_DEN},
		{input: "HOU", expected: TEAM_HOU},
		{input: "IND", expected: TEAM_IND},
		{input: "JAC", expected: TEAM_JAC},
		{input: "KCC", expected: TEAM_KCC},
		{input: "LVR", expected: TEAM_LVR},
		{input: "LAC", expected: TEAM_LAC},
		{input: "MIA", expected: TEAM_MIA},
		{input: "NEP", expected: TEAM_NEP},
		{input: "NYJ", expected: TEAM_NYJ},
		{input: "PIT", expected: TEAM_PIT},
		{input: "TEN", expected: TEAM_TEN},

		// Short names
		{input: "gb", expected: TEAM_GBP},
		{input: "lv", expected: TEAM_LVR},
		{input: "kc", expected: TEAM_KCC},
		{input: "ne", expected: TEAM_NEP},
		{input: "no", expected: TEAM_NOS},
		{input: "sf", expected: TEAM_SFO},
		{input: "tb", expected: TEAM_TBB},

		// mascot
		{input: "Panthers", expected: TEAM_CAR},
		{input: "Saints", expected: TEAM_NOS},
		{input: "Seahawks", expected: TEAM_SEA},
		{input: "Bangals", expected: TEAM_CIN},
		{input: "Dolphins", expected: TEAM_MIA},

		// location
		{input: "Dallas", expected: TEAM_DAL},
		{input: "Washington", expected: TEAM_WAS},
		{input: "Denver", expected: TEAM_DEN},

		// nicknames
		{input: "Philly", expected: TEAM_PHI},
		{input: "niners", expected: TEAM_SFO},
		{input: "9ers", expected: TEAM_SFO},
		{input: "pats", expected: TEAM_NEP},
		{input: "INDY", expected: TEAM_IND},

		// Unknown
		{input: "Puyallup", expected: TEAM_FA},
	}

	for _, tc := range tests {
		a := ParseTeam(tc.input)
		if !tc.expected.Equals(a) {
			t.Errorf("input: '%s', expected: '%s', got '%s'", tc.input, tc.expected, a)
		}
	}
}

func TestFriendly(t *testing.T) {
	tests := []struct {
		t    *NFLTeam
		want string
	}{
		{t: TEAM_SEA, want: "Seattle Seahawks"},
		{t: TEAM_FA, want: "FA"},
	}

	for _, tc := range tests {
		got := tc.t.Friendly()
		if tc.want != got {
			t.Errorf("expected: '%s', got: '%s'", tc.want, got)
		}
	}
}

func TestEquals(t *testing.T) {
	tests := []struct {
		a    *NFLTeam
		b    *NFLTeam
		want bool
	}{
		{a: TEAM_BAL, b: TEAM_BAL, want: true},
		{a: TEAM_SEA, b: TEAM_SFO, want: false},
		{a: TEAM_DAL, b: nil, want: false},
		{a: TEAM_SFO, b: TEAM_SFO, want: true},
	}

	for _, tc := range tests {
		got := tc.a.Equals(tc.b)
		if tc.want != got {
			t.Errorf("expected: '%v', got: '%v'", tc.want, got)
		}
	}
}
