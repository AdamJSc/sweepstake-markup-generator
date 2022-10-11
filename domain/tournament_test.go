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
		// TODO: add extra tournament validation tests
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
