package domain_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/sweepstake.adamjs.net/domain"
)

func TestMatchesCSVLoader_LoadMatches(t *testing.T) {
	tt := []struct {
		name        string
		testFile    string
		wantMatches domain.MatchCollection
		wantErr     error
	}{
		{
			name:     "valid matches csv must be loaded successfully",
			testFile: "matches_ok.csv",
			wantMatches: domain.MatchCollection{
				{
					ID:        "A1",
					Timestamp: time.Date(2018, 5, 26, 14, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "SWTFC"},
						Goals: 2,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						YellowCards: 2,
					},
					Winner: &domain.Team{
						ID: "SWTFC",
					},
					Completed: true,
				},
				{
					ID:        "A2",
					Timestamp: time.Date(2018, 5, 26, 19, 45, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "BPFC"},
						Goals:       1,
						YellowCards: 2,
					},
					Away: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "HUFC"},
						Goals: 1,
					},
					Completed: true,
				},
				{
					ID:        "B1",
					Timestamp: time.Date(2018, 5, 27, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DTFC"},
						YellowCards: 1,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DYFC"},
						Goals:       2,
						YellowCards: 1,
					},
					Winner: &domain.Team{
						ID: "DYFC",
					},
					Completed: true,
				},
				{
					ID:        "B2",
					Timestamp: time.Date(2018, 5, 27, 19, 45, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "SJRFC"},
						Goals: 2,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "STHFC"},
						YellowCards: 2,
					},
					Winner: &domain.Team{
						ID: "SJRFC",
					},
					Completed: true,
				},
				{
					ID:        "A3",
					Timestamp: time.Date(2018, 5, 28, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "BPFC"},
						Goals:       1,
						YellowCards: 2,
					},
					Away: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "SWTFC"},
						Goals: 1,
					},
					Completed: true,
				},
				{
					ID:        "A4",
					Timestamp: time.Date(2018, 5, 28, 19, 45, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "HUFC"},
						YellowCards: 1,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						Goals:       2,
						YellowCards: 1,
					},
					Winner: &domain.Team{
						ID: "PTFC",
					},
					Completed: true,
				},
				{
					ID:        "B3",
					Timestamp: time.Date(2018, 5, 29, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "DTFC"},
						Goals: 2,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SJRFC"},
						YellowCards: 2,
					},
					Winner: &domain.Team{
						ID: "DTFC",
					},
					Completed: true,
				},
				{
					ID:        "B4",
					Timestamp: time.Date(2018, 5, 29, 19, 45, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DYFC"},
						Goals:       1,
						YellowCards: 2,
					},
					Away: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "STHFC"},
						Goals: 1,
					},
					Completed: true,
				},
				{
					ID:        "A5",
					Timestamp: time.Date(2018, 5, 30, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "BPFC"},
						YellowCards: 1,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						Goals:       2,
						YellowCards: 1,
					},
					Winner: &domain.Team{
						ID: "PTFC",
					},
					Completed: true,
				},
				{
					ID:        "A6",
					Timestamp: time.Date(2018, 5, 30, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "HUFC"},
						Goals: 2,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SWTFC"},
						YellowCards: 2,
					},
					Winner: &domain.Team{
						ID: "HUFC",
					},
					Completed: true,
				},
				{
					ID:        "B5",
					Timestamp: time.Date(2018, 5, 31, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DTFC"},
						Goals:       1,
						YellowCards: 2,
					},
					Away: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "STHFC"},
						Goals: 1,
					},
					Completed: true,
				},
				{
					ID:        "B6",
					Timestamp: time.Date(2018, 5, 31, 15, 0, 0, 0, time.UTC),
					Stage:     domain.GroupStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DYFC"},
						YellowCards: 1,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SJRFC"},
						Goals:       2,
						YellowCards: 1,
					},
					Winner: &domain.Team{
						ID: "SJRFC",
					},
					Completed: true,
				},
				{
					ID:        "SF1",
					Timestamp: time.Date(2018, 6, 1, 15, 0, 0, 0, time.UTC),
					Stage:     domain.KnockoutStage,
					Home: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "PTFC"},
						Goals: 2,
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DTFC"},
						YellowCards: 2,
					},
					Winner: &domain.Team{
						ID: "PTFC",
					},
					Completed: true,
				},
				{
					ID:        "SF2",
					Timestamp: time.Date(2018, 6, 1, 15, 0, 0, 0, time.UTC),
					Stage:     domain.KnockoutStage,
					Home: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DYFC"},
						Goals:       1,
						YellowCards: 2,
					},
					Away: domain.MatchCompetitor{
						Team:  &domain.Team{ID: "BPFC"},
						Goals: 1,
					},
				},
				{
					ID:        "F",
					Timestamp: time.Date(2018, 6, 2, 15, 0, 0, 0, time.UTC),
					Stage:     domain.KnockoutStage,
					Home: domain.MatchCompetitor{
						Team: &domain.Team{ID: "PTFC"},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var testPath []string
			if tc.testFile != "" {
				testPath = []string{matchesDir, tc.testFile}
			}

			loader := newMatchesCSVLoader(testPath...)
			gotMatches, gotErr := loader.LoadMatches(nil)

			cmpDiff(t, tc.wantMatches, gotMatches)
			cmpError(t, tc.wantErr, gotErr)
		})
	}
}

func newMatchesCSVLoader(path ...string) *domain.MatchesCSVLoader {
	if len(path) > 0 {
		path = append([]string{testdataDir}, path...)
	}
	fullPath := filepath.Join(path...)
	return (&domain.MatchesCSVLoader{}).WithFileSystem(testdataFilesystem).WithPath(fullPath)
}
