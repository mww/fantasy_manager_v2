package controller

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mww/fantasy_manager_v2/model"
)

// Add a new rankings for players. This will parse the data from the reader (in CSV format) and
// create a new rankings data point. Returns the id of the new rankings and an error if there
// was one.
func (c *controller) AddRanking(ctx context.Context, r io.Reader, date time.Time) (int32, error) {
	playerRankings, err := c.getPlayerRankingMap(ctx, r)
	if err != nil {
		return 0, err
	}

	ranking, err := c.db.AddRanking(ctx, date, playerRankings)
	if err != nil {
		return 0, err
	}

	return ranking.ID, nil
}

func (c *controller) GetRanking(ctx context.Context, id int32) (*model.Ranking, error) {
	return c.db.GetRanking(ctx, id)
}

func (c *controller) DeleteRanking(ctx context.Context, id int32) error {
	return c.db.DeleteRanking(ctx, id)
}

func (c *controller) ListRankings(ctx context.Context) ([]model.Ranking, error) {
	return c.db.ListRankings(ctx)
}

func (c *controller) getPlayerRankingMap(ctx context.Context, r io.Reader) (map[string]int32, error) {
	reader, err := newFantasyProsCSVReader(r)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int32)

	for {
		line, err := reader.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if errors.Is(err, errUnusedPosition) {
				continue
			}
			return nil, err
		}

		query := fmt.Sprintf("%s team:%s pos:%s", line.name, line.team.String(), line.pos)
		matches, err := c.Search(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("error finding player %v: %w", line, err)
		}

		if len(matches) > 1 {
			return nil, fmt.Errorf("found more than 1 match for %v, got %d", line, len(matches))
		}

		if len(matches) == 0 {
			// Retry the query without the pos - sometimes fantasy pros have different positions for players
			query := fmt.Sprintf("%s team:%s", line.name, line.team.String())
			matches, err = c.Search(ctx, query)
			if err != nil {
				return nil, fmt.Errorf("error finding player %v: %w", line, err)
			}
		}

		if len(matches) != 1 {
			if line.rank > 500 {
				log.Printf("no match found for %v with rank %d, skipping", line, line.rank)
				continue
			}
			return nil, fmt.Errorf("did not find only a single player for %v, got %d", line, len(matches))
		}

		result[matches[0].ID] = line.rank
	}

	return result, nil
}

var errUnusedPosition = errors.New("unused position")

type fantasyprosCSVReader struct {
	csvReader *csv.Reader
	rankIdx   int
	nameIdx   int
	teamIdx   int
	posIdx    int
}

type csvLine struct {
	rank int32
	name string
	team *model.NFLTeam
	pos  model.Position
}

func (l *csvLine) String() string {
	return fmt.Sprintf("%d - %s %s %s", l.rank, l.name, l.team.String(), l.pos)
}

func newFantasyProsCSVReader(r io.Reader) (*fantasyprosCSVReader, error) {
	fp := &fantasyprosCSVReader{
		csvReader: csv.NewReader(r),
		rankIdx:   -1,
		nameIdx:   -1,
		teamIdx:   -1,
		posIdx:    -1,
	}

	header, err := fp.csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading fantasypros CSV file header: %v", err)
	}

	for i, p := range header {
		if p == "RK" {
			fp.rankIdx = i
		} else if p == "PLAYER NAME" {
			fp.nameIdx = i
		} else if p == "TEAM" {
			fp.teamIdx = i
		} else if p == "POS" {
			fp.posIdx = i
		}
	}

	if fp.rankIdx == -1 || fp.nameIdx == -1 || fp.teamIdx == -1 || fp.posIdx == -1 {
		return nil, fmt.Errorf("error finding required columns; rank: %d, name: %d, team: %d, pos: %d",
			fp.rankIdx, fp.nameIdx, fp.teamIdx, fp.posIdx)
	}

	return fp, nil
}

func (fp *fantasyprosCSVReader) readLine() (*csvLine, error) {
	record, err := fp.csvReader.Read()
	if errors.Is(err, io.EOF) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error reading line in rankings file (%v): %w", record, err)
	}

	line := csvLine{}

	rank, err := strconv.Atoi(record[fp.rankIdx])
	if err != nil {
		return nil, fmt.Errorf("error parsing ranking (%v): %w", record, err)
	}
	line.rank = int32(rank)

	line.name = trimNameSuffix(record[fp.nameIdx])

	t := record[fp.teamIdx]
	line.team = model.ParseTeam(t)
	if line.team == model.TEAM_FA && t != "FA" {
		return nil, fmt.Errorf("bad team name for %s", line.name)
	}

	line.pos = getPosition(record[fp.posIdx])
	if line.pos == model.POS_UNKNOWN {
		return nil, errUnusedPosition
	}

	return &line, nil
}

// Take a full name, like "Deebo Samuel Sr."" and return "Deebo Samuel".
func trimNameSuffix(fullName string) string {
	suffixList := []string{
		"Jr.",
		"Sr.",
		"II",
		"IV",
	}

	for _, s := range suffixList {
		fullName = strings.TrimSuffix(fullName, s)
	}

	return strings.TrimSpace(fullName)
}

var fpPosRegex = regexp.MustCompile(`(?P<pos>[A-Z]+)\d+`)

// Parse out the position from FantasyPros ranking file.
// Players are listed like WR1, RB7, QB12, K20, etc.
func getPosition(q string) model.Position {
	pos := model.POS_UNKNOWN
	m := fpPosRegex.FindStringSubmatch(q)
	if m != nil {
		p := m[fpPosRegex.SubexpIndex("pos")]
		pos = model.ParsePosition(p)
	}

	return pos
}
