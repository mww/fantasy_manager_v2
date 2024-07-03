package controller

import (
	"testing"

	"github.com/mww/fantasy_manager_v2/model"
)

func TestGetPositionFromQuery(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantQuery string
		wantPos   model.Position
	}{
		"position at end":    {input: "Tom Brady pos:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"upper case POS":     {input: "Tom Brady POS:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"position at start":  {input: "pos:QB Tom Brady", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"lower case pos":     {input: "DK Metcalf pos:wr", wantQuery: "DK Metcalf", wantPos: model.POS_WR},
		"position only":      {input: "pos:RB", wantQuery: "", wantPos: model.POS_RB},
		"no position":        {input: "TJ Hockenson", wantQuery: "TJ Hockenson", wantPos: model.POS_UNKNOWN},
		"unknown position":   {input: "Russell Wilson pos:QR", wantQuery: "Russell Wilson", wantPos: model.POS_UNKNOWN},
		"write out position": {input: "Tom Brady position:QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"space before :":     {input: "Tom Brady pos :QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"space after :":      {input: "Tom Brady pos: QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
		"spaces around :":    {input: "Tom Brady pos : QB", wantQuery: "Tom Brady", wantPos: model.POS_QB},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q, pos := getPositionFromQuery(tc.input)
			if tc.wantQuery != q {
				t.Errorf("query incorrect, wanted: '%s', got: '%s'", tc.wantQuery, q)
			}
			if tc.wantPos != pos {
				t.Errorf("position incorrect, wanted: '%s', got: '%s'", tc.wantPos, pos)
			}
		})
	}
}

func TestGetTeamFromQuery(t *testing.T) {
	tests := map[string]struct {
		input     string
		wantQuery string
		wantTeam  *model.NFLTeam
	}{
		"team at end":     {input: "AJ Brown team:PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"team at start":   {input: "team:PHI AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"uppercase TEAM":  {input: "TEAM:PHI AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"mascot":          {input: "team:eagles AJ Brown", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"city":            {input: "AJ Brown team:Philadelphia", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"nickname":        {input: "AJ Brown team:Philly", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"space before :":  {input: "AJ Brown team :PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"space after :":   {input: "AJ Brown team: PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"spaces around :": {input: "AJ Brown team : PHI", wantQuery: "AJ Brown", wantTeam: model.TEAM_PHI},
		"no team":         {input: "CeeDee Lamb", wantQuery: "CeeDee Lamb", wantTeam: nil},
		"bad team":        {input: "CeeDee Lamb team:puyallup", wantQuery: "CeeDee Lamb", wantTeam: nil},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q, team := getTeamFromQuery(tc.input)
			if tc.wantQuery != q {
				t.Errorf("query incorrect, wanted: '%s', got: '%s'", tc.wantQuery, q)
			}
			if tc.wantTeam != team {
				t.Errorf("team incorrect, wanted: '%s', got: '%s'", tc.wantTeam, team)
			}
		})
	}
}

// TODO: Add tests for Search().
