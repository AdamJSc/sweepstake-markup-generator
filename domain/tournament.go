package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
	"sync"
)

type Tournament struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageURL"`
	Teams    TeamCollection
	Matches  MatchCollection
	Template *template.Template
}

type TeamsLoader interface {
	LoadTeams(ctx context.Context) (TeamCollection, error)
}

type MatchesLoader interface {
	LoadMatches(ctx context.Context) (MatchCollection, error)
}

type TournamentFSLoader struct {
	fSys       fs.FS
	configPath string
	markupPath string
	tl         TeamsLoader
	ml         MatchesLoader
}

func (t *TournamentFSLoader) WithFileSystem(fSys fs.FS) *TournamentFSLoader {
	t.fSys = fSys
	return t
}

func (t *TournamentFSLoader) WithConfigPath(path string) *TournamentFSLoader {
	t.configPath = path
	return t
}

func (t *TournamentFSLoader) WithMarkupPath(path string) *TournamentFSLoader {
	t.markupPath = path
	return t
}

func (t *TournamentFSLoader) WithTeamsLoader(tl TeamsLoader) *TournamentFSLoader {
	t.tl = tl
	return t
}

func (t *TournamentFSLoader) WithMatchesLoader(ml MatchesLoader) *TournamentFSLoader {
	t.ml = ml
	return t
}

func (t *TournamentFSLoader) init() error {
	if t.fSys == nil {
		t.fSys = defaultFileSystem
	}

	if t.configPath == "" {
		return fmt.Errorf("config path: %w", ErrIsEmpty)
	}

	if t.markupPath == "" {
		return fmt.Errorf("config path: %w", ErrIsEmpty)
	}

	if t.tl == nil {
		return fmt.Errorf("teams loader: %w", ErrIsEmpty)
	}

	if t.ml == nil {
		return fmt.Errorf("matches loader: %w", ErrIsEmpty)
	}

	return nil
}

func (t *TournamentFSLoader) LoadTournament(ctx context.Context) (*Tournament, error) {
	if err := t.init(); err != nil {
		return nil, err
	}

	// read tournament config file
	b, err := readFile(t.fSys, t.configPath)
	if err != nil {
		return nil, err
	}

	// parse file contents
	tournament := &Tournament{}
	if err = json.Unmarshal(b, tournament); err != nil {
		return nil, fmt.Errorf("cannot unmarshal tournament: %w", err)
	}

	teams, err := t.tl.LoadTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load teams: %w", err)
	}

	matches, err := t.ml.LoadMatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load matches: %w", err)
	}

	tournament.Teams = teams
	tournament.Matches = matches

	// parse markup as template
	rawMarkup, err := readFile(t.fSys, t.markupPath)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("tpl").Parse(string(rawMarkup))
	if err != nil {
		return nil, fmt.Errorf("cannot parse template: %w", err)
	}

	tournament.Template = tpl

	mErr := NewMultiError()
	validateTournament(tournament, mErr)

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return tournament, nil
}

func validateTournament(tournament *Tournament, mErr MultiError) {
	tournament.ID = strings.Trim(tournament.ID, " ")
	tournament.Name = strings.Trim(tournament.Name, " ")
	tournament.ImageURL = strings.Trim(tournament.ImageURL, " ")

	if tournament.ID == "" {
		mErr.Add(fmt.Errorf("id: %w", ErrIsEmpty))
	}

	if tournament.Name == "" {
		mErr.Add(fmt.Errorf("name: %w", ErrIsEmpty))
	}

	if tournament.ImageURL == "" {
		mErr.Add(fmt.Errorf("image url: %w", ErrIsEmpty))
	}

	audit := &teamsAudit{teams: tournament.Teams}

	for idx, match := range tournament.Matches {
		matchNum := idx + 1
		mErrMatch := mErr.WithPrefix(fmt.Sprintf("match %d", matchNum))

		// enrich team entities based on existing ids
		if err := populateTeamByID(match.Home.Team, tournament.Teams); err != nil {
			mErrMatch.Add(fmt.Errorf("home: %w", err))
		}
		if err := populateTeamByID(match.Away.Team, tournament.Teams); err != nil {
			mErrMatch.Add(fmt.Errorf("away: %w", err))
		}
		if err := populateTeamByID(match.Winner, tournament.Teams); err != nil {
			mErrMatch.Add(fmt.Errorf("winner: %w", err))
		}

		// ensure that each tournament team appears at least once either home or away
		audit.ack(match.Home.Team)
		audit.ack(match.Away.Team)
	}

	audit.validate(mErr, true)
}

func populateTeamByID(team *Team, collection TeamCollection) error {
	if team == nil {
		return nil
	}

	if team.ID == "" {
		return nil
	}

	t := collection.GetByID(team.ID)
	if t == nil {
		return fmt.Errorf("team id '%s': %w", team.ID, ErrNotFound)
	}

	*team = *t

	return nil
}

type TournamentCollection []*Tournament

func (tc TournamentCollection) GetByID(id string) *Tournament {
	for _, tournament := range tc {
		if tournament != nil && tournament.ID == id {
			return tournament
		}
	}

	return nil
}

type TournamentLoader interface {
	LoadTournament(ctx context.Context) (*Tournament, error)
}

func NewTournamentCollection(ctx context.Context, loaders []TournamentLoader) (TournamentCollection, error) {
	var tournaments TournamentCollection

	for idx, loader := range loaders {
		tournament, err := loader.LoadTournament(ctx)
		if err != nil {
			return nil, fmt.Errorf("loader index %d: %w", idx, err)
		}

		tournaments = append(tournaments, tournament)
	}

	return validateTournaments(tournaments)
}

func validateTournaments(tournaments TournamentCollection) (TournamentCollection, error) {
	ids := &sync.Map{}
	mErr := NewMultiError()

	for _, tournament := range tournaments {
		// check if this tournament id already exists in the collection
		if _, ok := ids.Load(tournament.ID); ok {
			mErr.Add(fmt.Errorf("id '%s': %w", tournament.ID, ErrIsDuplicate))
		}
		ids.Store(tournament.ID, struct{}{})
	}

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return tournaments, nil
}
