package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"strings"
)

type Tournament struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageURL"`
	Teams    TeamCollection
	Matches  MatchCollection
}

type TeamsLoader interface {
	LoadTeams(ctx context.Context) (TeamCollection, error)
}

type MatchesLoader interface {
	LoadMatches(ctx context.Context) (MatchCollection, error)
}

type TournamentLoader struct {
	fSys fs.FS
	path string
	tl   TeamsLoader
	ml   MatchesLoader
}

func (t *TournamentLoader) WithFileSystem(fSys fs.FS) *TournamentLoader {
	t.fSys = fSys
	return t
}

func (t *TournamentLoader) WithPath(path string) *TournamentLoader {
	t.path = path
	return t
}

func (t *TournamentLoader) WithTeamsLoader(tl TeamsLoader) *TournamentLoader {
	t.tl = tl
	return t
}

func (t *TournamentLoader) WithMatchesLoader(ml MatchesLoader) *TournamentLoader {
	t.ml = ml
	return t
}

func (t *TournamentLoader) init() error {
	if t.fSys == nil {
		t.fSys = defaultFileSystem
	}

	if t.path == "" {
		return fmt.Errorf("path: %w", ErrIsEmpty)
	}

	if t.tl == nil {
		return fmt.Errorf("teams loader: %w", ErrIsEmpty)
	}

	if t.ml == nil {
		return fmt.Errorf("matches loader: %w", ErrIsEmpty)
	}

	return nil
}

func (t *TournamentLoader) LoadTournament(ctx context.Context) (*Tournament, error) {
	if err := t.init(); err != nil {
		return nil, err
	}

	// open tournament config file
	f, err := t.fSys.Open(t.path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	defer f.Close()

	// read file contents
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
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

	// TODO: populate matches with team entities from associated team collection
}
