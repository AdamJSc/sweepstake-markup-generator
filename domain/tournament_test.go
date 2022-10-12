package domain_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sweepstake.adamjs.net/domain"
)

var errSadTimes = errors.New("sad times :'(")

func TestTournamentLoader_LoadTournament(t *testing.T) {
	defaultMockTeamsLoader := newMockTeamsLoader(domain.TeamCollection{
		{ID: "123"}, {ID: "456"},
	}, nil)

	defaultMockMatchesLoader := newMockMatchesLoader(domain.MatchCollection{
		{ID: "654"}, {ID: "321"},
	}, nil)

	tt := []struct {
		name           string
		testFile       string
		teamsLoader    domain.TeamsLoader
		matchesLoader  domain.MatchesLoader
		wantTournament *domain.Tournament
		wantErr        error
	}{
		{
			name:     "valid tournament json must be loaded successfully",
			testFile: "tournament_ok.json",
			wantTournament: &domain.Tournament{
				ID:       "TestTourney1",
				Name:     "Test Tournament 1",
				ImageURL: "http://tourney.jpg",
				Teams: domain.TeamCollection{
					{ID: "123"}, {ID: "456"},
				},
				Matches: domain.MatchCollection{
					{ID: "654"}, {ID: "321"},
				},
			},
		},
		{
			name:    "empty path must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// testFile is empty
		},
		{
			name:     "non-existent path must produce the expected error",
			testFile: "non-existent.json",
			wantErr:  fs.ErrNotExist,
		},
		{
			name:     "invalid tournament format must produce the expected error",
			testFile: "tournament_unmarshalable.json",
			wantErr: fmt.Errorf("cannot unmarshal tournament: %w", &json.UnmarshalTypeError{
				Value:  "number",
				Struct: "Tournament",
				Type:   reflect.TypeOf("string"),
				Field:  "id",
			}),
		},
		{
			name:        "failure to load teams must produce the expected error",
			testFile:    "tournament_ok.json",
			teamsLoader: newMockTeamsLoader(nil, errSadTimes),
			wantErr:     errSadTimes,
		},
		{
			name:          "failure to load matches must produce the expected error",
			testFile:      "tournament_ok.json",
			matchesLoader: newMockMatchesLoader(nil, errSadTimes),
			wantErr:       errSadTimes,
		},
		{
			name:     "empty tournament must produce the expected error",
			testFile: "tournament_empty.json",
			wantErr: newMultiError([]string{
				"id: is empty",
				"name: is empty",
				"image url: is empty",
			}),
		},
		{
			name:     "teams that exist by id must be enriched successfully",
			testFile: "tournament_ok.json",
			teamsLoader: newMockTeamsLoader(domain.TeamCollection{
				{ID: "123", Name: "Team123", ImageURL: "http://team123.jpg"},
				{ID: "456", Name: "Team456", ImageURL: "http://team456.jpg"},
			}, nil),
			matchesLoader: newMockMatchesLoader(domain.MatchCollection{
				{
					Home:   domain.MatchCompetitor{Team: &domain.Team{ID: "123"}},
					Away:   domain.MatchCompetitor{Team: &domain.Team{ID: "456"}},
					Winner: &domain.Team{ID: "123"},
				},
			}, nil),
			wantTournament: &domain.Tournament{
				ID:       "TestTourney1",
				Name:     "Test Tournament 1",
				ImageURL: "http://tourney.jpg",
				Teams: domain.TeamCollection{
					{ID: "123", Name: "Team123", ImageURL: "http://team123.jpg"},
					{ID: "456", Name: "Team456", ImageURL: "http://team456.jpg"},
				},
				Matches: domain.MatchCollection{
					{
						Home:   domain.MatchCompetitor{Team: &domain.Team{ID: "123", Name: "Team123", ImageURL: "http://team123.jpg"}}, // fully-enriched team
						Away:   domain.MatchCompetitor{Team: &domain.Team{ID: "456", Name: "Team456", ImageURL: "http://team456.jpg"}}, // fully-enriched team
						Winner: &domain.Team{ID: "123", Name: "Team123", ImageURL: "http://team123.jpg"},                               // fully-enriched team
					},
				},
			},
		},
		{
			name:     "teams that do not exist by id must produce the expected error",
			testFile: "tournament_ok.json",
			matchesLoader: newMockMatchesLoader(domain.MatchCollection{
				{
					Home:   domain.MatchCompetitor{Team: &domain.Team{ID: "AAA"}},
					Away:   domain.MatchCompetitor{Team: &domain.Team{ID: "BBB"}},
					Winner: &domain.Team{ID: "CCC"},
				},
			}, nil),
			wantErr: newMultiError([]string{
				"match 1: home: team id 'AAA': not found",
				"match 1: away: team id 'BBB': not found",
				"match 1: winner: team id 'CCC': not found",
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var testPath []string
			if tc.testFile != "" {
				testPath = []string{tournamentsDir, tc.testFile}
			}

			teamsLoader := tc.teamsLoader
			if teamsLoader == nil {
				teamsLoader = defaultMockTeamsLoader
			}

			matchesLoader := tc.matchesLoader
			if matchesLoader == nil {
				matchesLoader = defaultMockMatchesLoader
			}

			loader := newTournamentLoader(testPath...).WithTeamsLoader(
				teamsLoader,
			).WithMatchesLoader(
				matchesLoader,
			)

			gotTournament, gotErr := loader.LoadTournament(ctx)

			cmpDiff(t, tc.wantTournament, gotTournament)
			cmpError(t, tc.wantErr, gotErr)
		})
	}
}

func newTournamentLoader(path ...string) *domain.TournamentLoader {
	if len(path) > 0 {
		path = append([]string{testdataDir}, path...)
	}
	fullPath := filepath.Join(path...)
	return (&domain.TournamentLoader{}).WithFileSystem(testdataFilesystem).WithPath(fullPath)
}

type mockTeamsLoader struct {
	teams domain.TeamCollection
	err   error
}

func (m *mockTeamsLoader) LoadTeams(_ context.Context) (domain.TeamCollection, error) {
	return m.teams, m.err
}

func newMockTeamsLoader(teams domain.TeamCollection, err error) *mockTeamsLoader {
	return &mockTeamsLoader{
		teams: teams,
		err:   err,
	}
}

type mockMatchesLoader struct {
	matches domain.MatchCollection
	err     error
}

func (m *mockMatchesLoader) LoadMatches(_ context.Context) (domain.MatchCollection, error) {
	return m.matches, m.err
}

func newMockMatchesLoader(matches domain.MatchCollection, err error) *mockMatchesLoader {
	return &mockMatchesLoader{
		matches: matches,
		err:     err,
	}
}
