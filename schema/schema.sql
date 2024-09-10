CREATE TABLE IF NOT EXISTS players (
    -- id == the sleeper player id
    id                varchar(16) PRIMARY KEY,
    yahoo_id          varchar(16),
    name_first        varchar(64) NOT NULL,
    name_last         varchar(64) NOT NULL,
    nickname1         varchar(64),
    position          varchar(3),
    team              varchar(3),
    weight_lb         smallint,
    height_in         smallint,
    birth_date        date,
    rookie_year       date,
    years_exp         smallint,
    jersey_num        smallint,
    depth_chart_order smallint,
    college           varchar(64),
    active            boolean,
    created           timestamp with time zone DEFAULT (now() at time zone 'utc'),
    updated           timestamp with time zone,
    fts_player        tsvector GENERATED ALWAYS AS (to_tsvector(
        'english', name_first || ' ' || name_last || coalesce(nickname1, '')
    )) STORED
);

CREATE TABLE IF NOT EXISTS player_changes (
    id      bigserial PRIMARY KEY,
    player  varchar(16) REFERENCES players(id),
    created timestamp DEFAULT (now() at time zone 'utc'),
    prop    varchar(32) NOT NULL,
    old     text NOT NULL,
    new     text NOT NULL
);

-- metadata about a ranking and a way to link all of the individual player
-- rankings together.
CREATE TABLE IF NOT EXISTS rankings (
    id           serial PRIMARY KEY,
    ranking_date timestamp with time zone NOT NULL,
    created      timestamp with time zone DEFAULT (now() at time zone 'utc')
);

-- The individual player rankings at a point in time.
CREATE TABLE IF NOT EXISTS player_rankings (
    ranking_id serial REFERENCES rankings(id),
    player_id  varchar(16) REFERENCES players(id),
    ranking    integer NOT NULL,
    PRIMARY KEY (ranking_id, player_id)
);

CREATE TABLE IF NOT EXISTS leagues (
    id          serial PRIMARY KEY,
    platform    varchar(16) NOT NULL, -- Where the league is hosted, sleeper, yahoo, etc.
    external_id varchar(64) NOT NULL, -- The id assigned by the platform. It is opaque to fantasy manager.
    name        varchar(64) NOT NULL,
    year        varchar(4) NOT NULL, -- The year of the league - YYYY. This is for systems where a new league id is generated each season.
    archived    boolean DEFAULT false,
    created     timestamp with time zone DEFAULT (now() at time zone 'utc')
);

CREATE TABLE IF NOT EXISTS tokens (
    league_id     serial REFERENCES leagues(id),
    access_token  text,
    refresh_token text,
    expires       timestamp with time zone,
    PRIMARY KEY (league_id)
);

CREATE TABLE IF NOT EXISTS league_managers (
    league_id    serial REFERENCES leagues(id),
    external_id  varchar(64) NOT NULL,
    team_name    varchar(64) NOT NULL,
    manager_name varchar(64),
    join_key     varchar(32), -- A value used help join different bits of data. e.g. sleeper uses "roster_id" in weekly scores.
    created      timestamp with time zone DEFAULT (now() at time zone 'utc'),
    PRIMARY KEY (league_id, external_id)
);

-- Keep track of how many fantasy points each player scored, by league and week.
-- The score is saved as 1/1000th of a point. So a score of 16.46 is saved as 16460.
-- Since most platforms only score to the 1/10th of a point, this should give room to grow. 
CREATE TABLE IF NOT EXISTS player_scores (
    player_id varchar(16) REFERENCES players(id),
    league_id serial REFERENCES leagues(id),
    week      smallint NOT NULL,
    score     integer NOT NULL,
    PRIMARY KEY (player_id, league_id, week)
);

CREATE SEQUENCE IF NOT EXISTS match_ids AS integer;

-- One half of a match up. This only has the score for one team in the match. The other
-- team will have the same match_id so they can be easily joined.
-- Like in player_points, score are staved as 1/1000th of a point.
CREATE TABLE IF NOT EXISTS team_results (
    id             serial PRIMARY KEY,
    league_id      serial REFERENCES leagues(id),
    week           smallint NOT NULL,
    match_id       serial NOT NULL, -- from the match_ids sequence
    team           varchar(64) NOT NULL,
    score          integer NOT NULL,
    FOREIGN KEY (league_id, team) REFERENCES league_managers(league_id, external_id)
);

-- These are instances of power rankings.
CREATE TABLE IF NOT EXISTS power_rankings (
    id         serial PRIMARY KEY,
    league_id  serial REFERENCES leagues(id),
    ranking_id serial REFERENCES rankings(id), -- Which set of rankings were used to calculate these rankings
    week       smallint, -- If set week will be used to determine win/loss records and streaks as they apply to power rankings 
    created    timestamp with time zone DEFAULT (now() at time zone 'utc')
);

-- These are the individual team results for a specific power ranking
CREATE TABLE IF NOT EXISTS team_power_rankings (
    power_ranking_id     serial REFERENCES power_rankings(id),
    league_id            serial REFERENCES leagues(id),
    team                 varchar(64),
    rank                 smallint NOT NULL, -- what ranking did the power ranking algorithm assign the team
    rank_change          smallint, -- how did the ranking change from the previous ranking?
    total_score          integer NOT NULL, -- how many points the power ranking algorithm assigned the team
    roster_score         integer, -- the portion of the total score from the roster
    record_score         integer, -- the portion of the total score from a team's record
    streak_score         integer, -- the portion of the total score from a team's current win/loss streak
    points_for_score     integer, -- the portion of the total score calculated by points scored
    points_against_score integer, -- the portion of the total score calculated by the points against
    PRIMARY KEY (power_ranking_id, league_id, team),
    FOREIGN KEY (league_id, team) REFERENCES league_managers(league_id, external_id)
);

CREATE TABLE IF NOT EXISTS power_rankings_rosters (
    power_ranking_id serial REFERENCES power_rankings(id),
    league_id        serial REFERENCES leagues(id),
    team             varchar(64),
    player_id        varchar(16) REFERENCES players(id),
    nfl_team         varchar(3), -- NFL team of the player, since teams change often
    player_rank      integer NOT NULL,
    -- The number of points assigned for this player.
    -- Non-starters only have a portion of their score used here.
    player_points    integer NOT NULL,
    starter          boolean NOT NULL,
    PRIMARY KEY (power_ranking_id, league_id, team, player_id),
    FOREIGN KEY (league_id, team) REFERENCES league_managers(league_id, external_id)
);

CREATE INDEX IF NOT EXISTS player_name_idx ON players USING gin(fts_player);
CREATE INDEX IF NOT EXISTS player_yahoo_id_idx ON players(yahoo_id);
CREATE INDEX IF NOT EXISTS player_change_idx ON player_changes(player, created DESC);
