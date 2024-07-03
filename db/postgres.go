package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/itbasis/go-clock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mww/fantasy_manager_v2/model"
)

var (
	ErrPlayerNotFound error = errors.New("player not found")
)

func New(ctx context.Context, connString string, clock clock.Clock) (DB, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &postgresDB{pool: pool, clock: clock}, nil
}

type postgresDB struct {
	pool  *pgxpool.Pool
	clock clock.Clock
}

func (db *postgresDB) GetPlayer(ctx context.Context, id string) (*model.Player, error) {
	p, err := db.getPlayer(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPlayerNotFound
		}
		return nil, err
	}
	return p, nil
}

func (db *postgresDB) SavePlayer(ctx context.Context, p *model.Player) error {
	old, err := db.getPlayer(ctx, p.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// This is an insert
			err := db.insertPlayer(ctx, p)
			if err != nil {
				return fmt.Errorf("error inserting player: %w", err)
			}
			return nil
		}

		return fmt.Errorf("error reading player at start of SavePlayer(): %w", err)
	}

	// This is an update, see what, if anything changed
	changes, err := db.calculateChanges(old, p)
	if err != nil {
		return fmt.Errorf("error calculating changes: %w", err)
	}
	if len(changes) > 0 {
		return db.updatePlayer(ctx, p, changes)
	}
	return nil
}

func (db *postgresDB) Search(ctx context.Context, q string, pos model.Position, team *model.NFLTeam) ([]model.Player, error) {
	const query = `SELECT id, yahoo_id, name_first, name_last, nickname1,
				  		position, team, weight_lb, height_in, birth_date,
						rookie_year, years_exp, jersey_num, depth_chart_order,
						college, active, created, updated
					FROM players WHERE fts_player @@ websearch_to_tsquery(@q)
						AND team ILIKE @team
						AND position ILIKE @pos`

	const teamAndPosQuery = `SELECT id, yahoo_id, name_first, name_last, nickname1,
					    		position, team, weight_lb, height_in, birth_date,
					  			rookie_year, years_exp, jersey_num, depth_chart_order,
					  			college, active, created, updated
				  			FROM players WHERE team ILIKE @team AND position ILIKE @pos`

	teamQ := "%"
	if team != nil {
		teamQ = team.String()
	}
	posQ := "%"
	if pos != model.POS_UNKNOWN {
		posQ = string(pos)
	}

	args := pgx.NamedArgs{
		"q":    q,
		"team": teamQ,
		"pos":  posQ,
	}

	qq := query
	if q == "" {
		qq = teamAndPosQuery
	}
	rows, err := db.pool.Query(ctx, qq, args)
	if err != nil {
		return nil, fmt.Errorf("error running search query: %w", err)
	}

	results := make([]model.Player, 0, 8)
	for rows.Next() {
		p, err := scanPlayer(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *p)
	}

	return results, nil
}

func (db *postgresDB) getPlayer(ctx context.Context, id string) (*model.Player, error) {
	const query = `SELECT id, yahoo_id, name_first, name_last, nickname1,
				  		position, team, weight_lb, height_in, birth_date,
						rookie_year, years_exp, jersey_num, depth_chart_order,
						college, active, created, updated
					FROM players WHERE id=@id`

	args := pgx.NamedArgs{
		"id": id,
	}
	row := db.pool.QueryRow(ctx, query, args)
	result, err := scanPlayer(row)
	if err != nil {
		return nil, fmt.Errorf("error scanning player %s: %w", id, err)
	}

	changes, err := db.getChangesByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error looking up player changes for %s: %w", id, err)
	}
	result.Changes = changes

	return result, nil
}

func scanPlayer(row pgx.Row) (*model.Player, error) {
	var result model.Player
	var pos DBPosition
	var team DBNFLTeam
	var yahooID, nickname1, college sql.NullString
	var birthDate, rookieYear pgtype.Date
	var created, updated pgtype.Timestamptz
	err := row.Scan(
		&result.ID,
		&yahooID,
		&result.FirstName,
		&result.LastName,
		&nickname1,
		&pos,
		&team,
		&result.Weight,
		&result.Height,
		&birthDate,
		&rookieYear,
		&result.YearsExp,
		&result.Jersey,
		&result.DepthChartOrder,
		&college,
		&result.Active,
		&created,
		&updated)

	if err != nil {
		return nil, err
	}

	result.Position = pos.position
	result.Team = team.team
	result.YahooID = valueOrEmpty(yahooID)
	result.Nickname1 = valueOrEmpty(nickname1)
	result.College = valueOrEmpty(college)
	result.BirthDate = birthDate.Time
	result.RookieYear = rookieYear.Time
	result.Created = created.Time
	result.Updated = updated.Time

	return &result, nil
}

func (db *postgresDB) getChangesByID(ctx context.Context, id string) ([]model.Change, error) {
	const query = `SELECT created, prop, old, new FROM player_changes WHERE player=@id ORDER BY created DESC`

	args := pgx.NamedArgs{
		"id": id,
	}
	rows, err := db.pool.Query(ctx, query, args)
	if err != nil {
		return nil, err
	}
	changes := make([]model.Change, 0, 16)
	for rows.Next() {
		var created pgtype.Timestamptz
		c := model.Change{}
		err := rows.Scan(&created, &c.PropertyName, &c.OldValue, &c.NewValue)
		if err != nil {
			return nil, fmt.Errorf("error scanning player change: %v", err)
		}
		c.Time = created.Time

		changes = append(changes, c)
	}

	return changes, nil
}

func (db *postgresDB) insertPlayer(ctx context.Context, p *model.Player) error {
	if p == nil {
		return errors.New("insertPlayer - player is nil")
	}
	const query = `INSERT INTO players (
		id,
		yahoo_id,
		name_first,
		name_last,
		nickname1,
		position,
		team,
		weight_lb,
		height_in,
		birth_date,
		rookie_year,
		years_exp,
		jersey_num,
		depth_chart_order,
		college,
		active
	) VALUES (
		@id,
		@yahooID,
		@nameFirst,
		@nameLast,
		@nickname1,
		@position,
		@team,
		@weight,
		@height,
		@birthDate,
		@rookieYear,
		@yearsExp,
		@jerseyNum,
		@depthChartOrder,
		@college,
		@active
	)`

	args := namedArgsForPlayer(p, db.clock)
	_, err := db.pool.Exec(ctx, query, args)
	if err != nil {
		return fmt.Errorf("error inserting player(%s): %w", p.ID, err)
	}
	return nil
}

func (db *postgresDB) updatePlayer(ctx context.Context, p *model.Player, changes []model.Change) error {
	const update = `UPDATE players
		SET yahoo_id=@yahooID,
			name_first=@nameFirst,
			name_last=@nameLast,
			nickname1=@nickname1,
			position=@position,
			team=@team,
			weight_lb=@weight,
			height_in=@height,
			birth_date=@birthDate,
			rookie_year=@rookieYear,
			years_exp=@yearsExp,
			jersey_num=@jerseyNum,
			depth_chart_order=@depthChartOrder,
			college=@college,
			active=@active,
			updated=@updated
		WHERE id=@id`

	const insertChange = `INSERT INTO player_changes(
		player,
		prop,
		old,
		new
	) VALUES (
		@playerId,
		@prop,
		@old,
		@new
	)`

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	args := namedArgsForPlayer(p, db.clock)
	_, err = tx.Exec(ctx, update, args)
	if err != nil {
		return fmt.Errorf("error updating player (%s): %w", p.ID, err)
	}

	for _, change := range changes {
		args := namedArgsForPlayerChange(p.ID, &change)
		_, err = tx.Exec(ctx, insertChange, args)
		if err != nil {
			return fmt.Errorf("error inserting player change: %w", err)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("error commiting player transaction: %w", err)
	}

	p.Changes = append(p.Changes, changes...)
	slices.SortFunc(p.Changes, func(a, b model.Change) int {
		return b.Time.Compare(a.Time)
	})

	return nil
}

func (db *postgresDB) calculateChanges(p1, p2 *model.Player) ([]model.Change, error) {
	changes := make([]model.Change, 0, 1)

	changes = checkChange(changes, db.clock, "YahooId", p1.YahooID, p2.YahooID)
	changes = checkChange(changes, db.clock, "FirstName", p1.FirstName, p2.FirstName)
	changes = checkChange(changes, db.clock, "LastName", p1.LastName, p2.LastName)
	changes = checkChange(changes, db.clock, "Position", string(p1.Position), string(p2.Position))
	changes = checkChange(changes, db.clock, "Team", p1.Team.String(), p2.Team.String())
	changes = checkChangeInt(changes, db.clock, "Weight", p1.Weight, p2.Weight)
	changes = checkChangeInt(changes, db.clock, "Height", p1.Height, p2.Height)
	changes = checkChange(changes, db.clock, "BirthDate", p1.FormattedBirthDate(), p2.FormattedBirthDate())
	changes = checkChange(changes, db.clock, "RookieYear", p1.FormattedRookieYear(), p2.FormattedRookieYear())
	changes = checkChangeInt(changes, db.clock, "YearsExp", p1.YearsExp, p2.YearsExp)
	changes = checkChangeInt(changes, db.clock, "Jersey", p1.Jersey, p2.Jersey)
	changes = checkChangeInt(changes, db.clock, "DepthChartOrder", p1.DepthChartOrder, p2.DepthChartOrder)
	changes = checkChange(changes, db.clock, "College", p1.College, p2.College)
	changes = checkChange(changes, db.clock, "Active", fmt.Sprintf("%v", p1.Active), fmt.Sprintf("%v", p2.Active))

	// Nickname is special because it isn't part of the sleeper data, therefore
	// don't delete if it exists in p1 but not in p2.
	if p1.Nickname1 != "" && p2.Nickname1 == "" {
		changes = checkChange(changes, db.clock, "Nickname1", p1.Nickname1, p2.Nickname1)
	}
	return changes, nil
}

func checkChange(changes []model.Change, clock clock.Clock, prop, old, new string) []model.Change {
	if old != new {
		c := model.Change{
			Time:         clock.Now().UTC(),
			PropertyName: prop,
			OldValue:     old,
			NewValue:     new,
		}
		changes = append(changes, c)
	}
	return changes
}

func checkChangeInt(changes []model.Change, clock clock.Clock, prop string, old, new int) []model.Change {
	return checkChange(changes, clock, prop, fmt.Sprintf("%d", old), fmt.Sprintf("%d", new))
}

func namedArgsForPlayer(p *model.Player, clock clock.Clock) pgx.NamedArgs {
	return pgx.NamedArgs{
		"id": p.ID,
		"yahooID": sql.NullString{
			String: p.YahooID,
			Valid:  p.YahooID != "",
		},
		"nameFirst": p.FirstName,
		"nameLast":  p.LastName,
		"nickname1": sql.NullString{
			String: p.Nickname1,
			Valid:  p.Nickname1 != "",
		},
		"position": &DBPosition{position: p.Position},
		"team":     &DBNFLTeam{team: p.Team},
		"weight":   p.Weight,
		"height":   p.Height,
		"birthDate": pgtype.Date{
			Time:  p.BirthDate,
			Valid: !p.BirthDate.IsZero(),
		},
		"rookieYear": pgtype.Date{
			Time:  p.RookieYear,
			Valid: !p.RookieYear.IsZero(),
		},
		"yearsExp":        p.YearsExp,
		"jerseyNum":       p.Jersey,
		"depthChartOrder": p.DepthChartOrder,
		"college": sql.NullString{
			String: p.College,
			Valid:  p.College != "",
		},
		"active": p.Active,
		"updated": pgtype.Timestamptz{
			Time:             clock.Now().UTC(),
			InfinityModifier: pgtype.Finite,
			Valid:            true,
		},
	}
}

func namedArgsForPlayerChange(playerId string, c *model.Change) pgx.NamedArgs {
	return pgx.NamedArgs{
		"playerId": playerId,
		"prop":     c.PropertyName,
		"old":      c.OldValue,
		"new":      c.NewValue,
	}
}

func valueOrEmpty(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

type DBPosition struct {
	position model.Position
}

func (p *DBPosition) ScanText(v pgtype.Text) error {
	p.position = model.ParsePosition(v.String)
	return nil
}

func (p *DBPosition) TextValue() (pgtype.Text, error) {
	return pgtype.Text{
		String: string(p.position),
		Valid:  true,
	}, nil
}

type DBNFLTeam struct {
	team *model.NFLTeam
}

func (t *DBNFLTeam) ScanText(v pgtype.Text) error {
	t.team = model.ParseTeam(v.String)
	return nil
}

func (t *DBNFLTeam) TextValue() (pgtype.Text, error) {
	return pgtype.Text{
		String: t.team.String(),
		Valid:  true,
	}, nil
}
