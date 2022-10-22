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

const (
	tournamentConfigOkFilename = "tournament_config_ok.json"
	tournamentMarkupOkFilename = "tournament_markup_ok.gohtml"
)

var errSadTimes = errors.New("sad times :'(")

func TestTournamentFSLoader_LoadTournament(t *testing.T) {
	defaultTeamCollection := domain.TeamCollection{
		{ID: "123"}, {ID: "456"},
	}

	defaultMatchCollection := domain.MatchCollection{
		{ID: "654"},
		{
			ID: "321",
			Home: domain.MatchCompetitor{
				Team: &domain.Team{ID: "123"},
			},
			Away: domain.MatchCompetitor{
				Team: &domain.Team{ID: "456"},
			},
		},
	}

	defaultMockTeamsLoader := newMockTeamsLoader(defaultTeamCollection, nil)
	defaultMockMatchesLoader := newMockMatchesLoader(defaultMatchCollection, nil)

	tt := []struct {
		name           string
		configFilename string
		markupFilename string
		teamsLoader    domain.TeamsLoader
		matchesLoader  domain.MatchesLoader
		wantTournament *domain.Tournament
		wantErr        error
	}{
		{
			name:           "valid tournament json must be loaded successfully",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  defaultMockMatchesLoader,
			wantTournament: &domain.Tournament{
				ID:       "TestTourney1",
				Name:     "Test Tournament 1",
				ImageURL: "http://tourney.jpg",
				Teams:    defaultTeamCollection,
				Matches:  defaultMatchCollection,
				Template: parseTemplate(t, "<h1>Hello World</h1>"),
			},
		},
		{
			name:    "empty config path must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// configFilename is empty
		},
		{
			name:           "empty markup path must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			wantErr:        domain.ErrIsEmpty,
			// markupFilename is empty
		},
		{
			name:           "empty teams loader must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			wantErr:        domain.ErrIsEmpty,
			// teamsLoader is empty
		},
		{
			name:           "empty matches loader must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			wantErr:        domain.ErrIsEmpty,
			// matchesLoader is empty
		},
		{
			name:           "non-existent config path must produce the expected error",
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  defaultMockMatchesLoader,
			configFilename: "non-existent.json",
			markupFilename: tournamentMarkupOkFilename,
			wantErr:        fs.ErrNotExist,
		},
		{
			name:           "invalid tournament format must produce the expected error",
			configFilename: "tournament_config_unmarshalable.json",
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  defaultMockMatchesLoader,
			wantErr: fmt.Errorf("cannot unmarshal tournament: %w", &json.UnmarshalTypeError{
				Value:  "number",
				Struct: "Tournament",
				Type:   reflect.TypeOf("string"),
				Field:  "id",
			}),
		},
		{
			name:           "failure to load teams must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    newMockTeamsLoader(nil, errSadTimes),
			matchesLoader:  defaultMockMatchesLoader,
			wantErr:        errSadTimes,
		},
		{
			name:           "failure to load matches must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  newMockMatchesLoader(nil, errSadTimes),
			wantErr:        errSadTimes,
		},
		{
			name:           "empty tournament must produce the expected error",
			configFilename: "tournament_config_empty.json",
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  defaultMockMatchesLoader,
			wantErr: newMultiError([]string{
				"id: is empty",
				"name: is empty",
				"image url: is empty",
			}),
		},
		{
			name:           "teams that exist by id must be enriched successfully",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
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
				Template: parseTemplate(t, "<h1>Hello World</h1>"),
			},
		},
		{
			name:           "teams that do not exist by id must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader: newMockMatchesLoader(domain.MatchCollection{
				{
					Home: domain.MatchCompetitor{Team: &domain.Team{ID: "123"}}, // included in default team collection
					Away: domain.MatchCompetitor{Team: &domain.Team{ID: "456"}}, // included in default team collection
				},
				{
					Home:   domain.MatchCompetitor{Team: &domain.Team{ID: "AAA"}},
					Away:   domain.MatchCompetitor{Team: &domain.Team{ID: "BBB"}},
					Winner: &domain.Team{ID: "CCC"},
				},
			}, nil),
			wantErr: newMultiError([]string{
				"match 2: home: team id 'AAA': not found",
				"match 2: away: team id 'BBB': not found",
				"match 2: winner: team id 'CCC': not found",
			}),
		},
		{
			name:           "teams that are not accounted for within any matches must produce the expected error",
			configFilename: tournamentConfigOkFilename,
			markupFilename: tournamentMarkupOkFilename,
			teamsLoader:    defaultMockTeamsLoader,
			matchesLoader:  newMockMatchesLoader(domain.MatchCollection{}, nil),
			wantErr: newMultiError([]string{
				"team id '123': count 0", // first team in team collection
				"team id '456': count 0", // second team in team collection
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var configPath, markupPath string
			if tc.configFilename != "" {
				configPath = filepath.Join(testdataDir, tournamentsDir, tc.configFilename)
			}
			if tc.markupFilename != "" {
				markupPath = filepath.Join(testdataDir, tournamentsDir, tc.markupFilename)
			}

			loader := (&domain.TournamentFSLoader{}).
				WithFileSystem(testdataFilesystem).
				WithConfigPath(configPath).
				WithMarkupPath(markupPath).
				WithTeamsLoader(tc.teamsLoader).
				WithMatchesLoader(tc.matchesLoader)

			gotTournament, gotErr := loader.LoadTournament(ctx)

			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantTournament, gotTournament)
		})
	}
}

func TestTournamentCollection_GetByID(t *testing.T) {
	tournamentA1 := &domain.Tournament{
		ID:       "tourneyA",
		Name:     "TourneyA1",
		ImageURL: "http://tourney-a1.jpg",
	}

	tournamentB := &domain.Tournament{
		ID:       "tourneyB",
		Name:     "TourneyB",
		ImageURL: "http://tourney-b.jpg",
	}

	tournamentA2 := &domain.Tournament{
		ID:       "tourneyA",
		Name:     "TourneyA2",
		ImageURL: "http://tourney-a2.jpg",
	}

	collection := domain.TournamentCollection{
		tournamentA1,
		tournamentB,
		tournamentA2, // duplicate id, should never be returned (tournamentA1 should match first)
	}

	tt := []struct {
		name           string
		id             string
		wantTournament *domain.Tournament
	}{
		{
			name:           "duplicate tournament id must return first matched item",
			id:             "tourneyA",
			wantTournament: tournamentA1,
		},
		{
			name:           "unique tournament id must return only matching item",
			id:             "tourneyB",
			wantTournament: tournamentB,
		},
		{
			name: "non-matching item must return nil",
			id:   "tourneyC",
			// want nil team
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotTeam := collection.GetByID(tc.id)
			cmpDiff(t, tc.wantTournament, gotTeam)
		})
	}
}

func TestNewTournamentCollection(t *testing.T) {
	tt := []struct {
		name           string
		loaders        []domain.TournamentLoader
		wantCollection domain.TournamentCollection
		wantErr        error
	}{
		{
			name: "valid loaders must be processed successfully",
			loaders: []domain.TournamentLoader{
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament1",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament2",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament3",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament4",
				}, nil),
			},
			wantCollection: domain.TournamentCollection{
				{ID: "tournament1"},
				{ID: "tournament2"},
				{ID: "tournament3"},
				{ID: "tournament4"},
			},
		},
		{
			name: "processing loaders that return errors must produce the expected error",
			loaders: []domain.TournamentLoader{
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament1",
				}, nil),
				newMockTournamentLoader(nil, fmt.Errorf("tournament2: %w", errSadTimes)),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament3",
				}, nil),
			},
			wantErr: errSadTimes,
		},
		{
			name: "duplicate tournament ids must produce the expected error",
			loaders: []domain.TournamentLoader{
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament1",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament2",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament1",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament2",
				}, nil),
				newMockTournamentLoader(&domain.Tournament{
					ID: "tournament3",
				}, nil),
			},
			wantErr: newMultiError([]string{
				"id 'tournament1': is duplicate",
				"id 'tournament2': is duplicate",
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			gotCollection, gotErr := domain.NewTournamentCollection(ctx, tc.loaders)

			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantCollection, gotCollection)
		})
	}
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

type mockTournamentLoader struct {
	tournament *domain.Tournament
	err        error
}

func (m *mockTournamentLoader) LoadTournament(_ context.Context) (*domain.Tournament, error) {
	return m.tournament, m.err
}

func newMockTournamentLoader(tournament *domain.Tournament, err error) *mockTournamentLoader {
	return &mockTournamentLoader{
		tournament: tournament,
		err:        err,
	}
}
