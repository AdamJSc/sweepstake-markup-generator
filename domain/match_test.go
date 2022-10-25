package domain_test

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/sweepstake.adamjs.net/domain"
)

func TestMatchCollection_GetByID(t *testing.T) {
	matchA1 := &domain.Match{
		ID: "matchA",
	}

	matchB := &domain.Match{
		ID: "matchB",
	}

	matchA2 := &domain.Match{
		ID: "matchA",
	}

	collection := domain.MatchCollection{
		matchA1,
		matchB,
		matchA2, // duplicate id, should never be returned (matchA1 should match first)
	}

	tt := []struct {
		name      string
		id        string
		wantMatch *domain.Match
	}{
		{
			name:      "duplicate match id must return first matched item",
			id:        "matchA",
			wantMatch: matchA1,
		},
		{
			name:      "unique match id must return only matching item",
			id:        "matchB",
			wantMatch: matchB,
		},
		{
			name: "non-matching item must return nil",
			id:   "matchC",
			// want nil team
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotMatch := collection.GetByID(tc.id)
			cmpDiff(t, tc.wantMatch, gotMatch)
		})
	}
}

func TestMatchCollection_GetWinnerByMatchID(t *testing.T) {
	matchID := "test-match"

	team := &domain.Team{
		ID: "test-team",
	}

	tt := []struct {
		name            string
		matchCollection domain.MatchCollection
		wantTeam        *domain.Team
	}{
		{
			name: "existent match id and completed match and non-nil winner must produce the expected team",
			matchCollection: domain.MatchCollection{
				{
					ID:        "test-match",
					Completed: true,
					Winner:    team,
				},
			},
			wantTeam: team,
		},
		{
			name: "non-existent match id must return nil",
			matchCollection: domain.MatchCollection{
				{
					ID:        "not-test-match",
					Completed: true,
					Winner:    team,
				},
			},
			// wantTeam is nil
		},
		{
			name: "match that is not completed must return nil",
			matchCollection: domain.MatchCollection{
				{
					ID:     "test-match",
					Winner: team,
					// completed is false
				},
			},
			// wantTeam is nil
		},
		{
			name: "empty team must return nil",
			matchCollection: domain.MatchCollection{
				{
					ID:        "test-match",
					Completed: true,
					// winner is nil
				},
			},
			// wantTeam is nil
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotTeam := tc.matchCollection.GetWinnerByMatchID(matchID)
			cmpDiff(t, tc.wantTeam, gotTeam)
		})
	}
}

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
						Team:     &domain.Team{ID: "SWTFC"},
						Goals:    2,
						OwnGoals: []domain.MatchEvent{{Name: "O'Brien", Minute: 12}},
						RedCards: []domain.MatchEvent{{Name: "Prichard", Minute: 22}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						YellowCards: 2,
						OwnGoals:    []domain.MatchEvent{{Name: "Thiessen", Minute: 54}},
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
						Team:     &domain.Team{ID: "HUFC"},
						Goals:    1,
						OwnGoals: []domain.MatchEvent{{Name: "Friend", Minute: 43}, {Name: "Jefferson", Minute: 89}},
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
						OwnGoals:    []domain.MatchEvent{{Name: "Johnson", Minute: 11}, {Name: "Smith", Minute: 34}},
						RedCards:    []domain.MatchEvent{{Name: "Isome", Minute: 25}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DYFC"},
						Goals:       2,
						YellowCards: 1,
						RedCards:    []domain.MatchEvent{{Name: "Reid-Cunningham", Minute: 56}},
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
						Team:     &domain.Team{ID: "SJRFC"},
						Goals:    2,
						OwnGoals: []domain.MatchEvent{{Name: "Jones", Minute: 7}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "STHFC"},
						YellowCards: 2,
						OwnGoals:    []domain.MatchEvent{{Name: "Moriarty", Minute: 21}},
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
						RedCards:    []domain.MatchEvent{{Name: "Sheahan", Minute: 8}},
					},
					Away: domain.MatchCompetitor{
						Team:     &domain.Team{ID: "SWTFC"},
						Goals:    1,
						OwnGoals: []domain.MatchEvent{{Name: "Racoosin", Minute: 33}, {Name: "Broadfoot", Minute: 90, Offset: 2}},
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
						OwnGoals:    []domain.MatchEvent{{Name: "Kenny", Minute: 65}, {Name: "Jensen", Minute: 80}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						Goals:       2,
						YellowCards: 1,
						RedCards:    []domain.MatchEvent{{Name: "Pesarin", Minute: 22}},
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
						Team:     &domain.Team{ID: "DTFC"},
						Goals:    2,
						OwnGoals: []domain.MatchEvent{{Name: "Scott", Minute: 45, Offset: 4}},
						RedCards: []domain.MatchEvent{{Name: "Neilson", Minute: 67}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SJRFC"},
						YellowCards: 2,
						OwnGoals:    []domain.MatchEvent{{Name: "Fillios", Minute: 89}},
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
						Team:     &domain.Team{ID: "STHFC"},
						Goals:    1,
						OwnGoals: []domain.MatchEvent{{Name: "Landenna", Minute: 20}, {Name: "Dongoski", Minute: 24}},
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
						OwnGoals:    []domain.MatchEvent{{Name: "Peterson", Minute: 9}, {Name: "Williamson", Minute: 33}},
						RedCards:    []domain.MatchEvent{{Name: "Wacquant", Minute: 11}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "PTFC"},
						Goals:       2,
						YellowCards: 1,
						RedCards:    []domain.MatchEvent{{Name: "Sewall", Minute: 32}},
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
						Team:     &domain.Team{ID: "HUFC"},
						Goals:    2,
						OwnGoals: []domain.MatchEvent{{Name: "McCartney", Minute: 12}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SWTFC"},
						YellowCards: 2,
						OwnGoals:    []domain.MatchEvent{{Name: "Margaitis", Minute: 59}},
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
						RedCards:    []domain.MatchEvent{{Name: "Bhide", Minute: 55}},
					},
					Away: domain.MatchCompetitor{
						Team:     &domain.Team{ID: "STHFC"},
						Goals:    1,
						OwnGoals: []domain.MatchEvent{{Name: "Daboni", Minute: 76}, {Name: "T.Wegman", Minute: 77}},
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
						OwnGoals:    []domain.MatchEvent{{Name: "Lennon", Minute: 1}, {Name: "Starr", Minute: 46}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "SJRFC"},
						Goals:       2,
						YellowCards: 1,
						RedCards:    []domain.MatchEvent{{Name: "Glover", Minute: 44}, {Name: "Litwin", Minute: 23}},
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
						Team:     &domain.Team{ID: "PTFC"},
						Goals:    2,
						OwnGoals: []domain.MatchEvent{{Name: "Harrison", Minute: 7}},
						RedCards: []domain.MatchEvent{{Name: "St.Martin", Minute: 13}},
					},
					Away: domain.MatchCompetitor{
						Team:        &domain.Team{ID: "DTFC"},
						YellowCards: 2,
						OwnGoals:    []domain.MatchEvent{{Name: "Bickmore", Minute: 41}},
						RedCards:    []domain.MatchEvent{{Name: "Kinnaman", Minute: 77}},
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
						Team:     &domain.Team{ID: "BPFC"},
						Goals:    1,
						OwnGoals: []domain.MatchEvent{{Name: "Lomeli", Minute: 67}, {Name: "Prichard", Minute: 89}},
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
		{
			name:    "empty path must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// testFile is empty
		},
		{
			name:     "non-existent path must produce the expected error",
			testFile: "non-existent.csv",
			wantErr:  fs.ErrNotExist,
		},
		{
			name:     "file with invalid number of row fields must produce the expected error",
			testFile: "matches_invalid_file.csv",
			wantErr:  errors.New("cannot read file: record on line 2: wrong number of fields"),
		},
		{
			name:     "empty file must produce the expected error",
			testFile: "matches_empty.csv",
			wantErr:  errors.New("cannot transform csv: rows 0: file must have header row and at least one more row"),
		},
		{
			name:     "file with only one row must produce the expected error",
			testFile: "matches_header_row_only.csv",
			wantErr:  errors.New("cannot transform csv: rows 1: file must have header row and at least one more row"),
		},
		{
			name:     "file with invalid header row must produce the expected error",
			testFile: "matches_invalid_header_row.csv",
			wantErr:  errors.New("cannot transform csv: invalid headers: header,row"),
		},
		{
			name:     "file with invalid timestamps must produce the expected error",
			testFile: "matches_rows_with_invalid_timestamp.csv",
			wantErr: fmt.Errorf("cannot transform csv: %w", newMultiError([]string{
				"row 1: invalid timestamp format: epic fail",
				"row 2: invalid timestamp format: sad 15:00",
				"row 3: invalid timestamp format: 02/06/2018 times",
			})),
		},
		{
			name:     "file with invalid stage must produce the expected error",
			testFile: "matches_rows_with_invalid_stage.csv",
			wantErr: fmt.Errorf("cannot transform csv: %w", newMultiError([]string{
				"row 1: invalid match stage: NOT_A_VALID_STAGE",
			})),
		},
		{
			name:     "file with invalid goals must produce the expected error",
			testFile: "matches_rows_with_invalid_goals.csv",
			wantErr: fmt.Errorf("cannot transform csv: %w", newMultiError([]string{
				`row 1: home goals: invalid int: strconv.Atoi: parsing "OH": invalid syntax`,
				`row 1: away goals: invalid int: strconv.Atoi: parsing "NO!": invalid syntax`,
			})),
		},
		{
			name:     "file with invalid yellow cards must produce the expected error",
			testFile: "matches_rows_with_invalid_yellow_cards.csv",
			wantErr: fmt.Errorf("cannot transform csv: %w", newMultiError([]string{
				`row 1: home yellow cards: invalid int: strconv.Atoi: parsing "OH": invalid syntax`,
				`row 1: away yellow cards: invalid int: strconv.Atoi: parsing "NO!": invalid syntax`,
			})),
		},
		{
			name:     "file with invalid match events must produce the expected error",
			testFile: "matches_rows_with_invalid_match_events.csv",
			wantErr: fmt.Errorf("cannot transform csv: %w", newMultiError([]string{
				`row 1: home own goals: first element must provide count of remaining elements`,
				`row 1: home red cards: first element must provide count of remaining elements`,
				`row 1: away own goals: first element must provide count of remaining elements`,
				`row 1: away red cards: first element must provide count of remaining elements`,
				`row 2: home own goals: must have 1 element`,
				`row 2: home red cards: must have 3 elements`,
				`row 2: away own goals: must have 2 elements`,
				`row 2: away red cards: must have 4 elements`,
				`row 3: home own goals: event 1: invalid format`,
				`row 3: home red cards: event 2: invalid format`,
				`row 3: away own goals: event 1: minute: invalid int: strconv.Atoi: parsing "invalidNumber": invalid syntax`,
				`row 3: away red cards: event 2: minute: invalid int: strconv.Atoi: parsing "invalidNumber": invalid syntax`,
				`row 4: away own goals: event 1: minute: must be greater than 0`,
				`row 5: home red cards: event 1: offset: invalid int: strconv.Atoi: parsing "invalidNumber": invalid syntax`,
				`row 6: away red cards: event 1: offset: must be greater than 0`,
			})),
		},

		{
			name:     "empty match id must produce the expected error",
			testFile: "matches_rows_with_missing_id.csv",
			wantErr: newMultiError([]string{
				`index 0: id: is empty`,
			}),
		},
		{
			name:     "empty timestamp must produce the expected error",
			testFile: "matches_rows_with_empty_timestamp.csv",
			wantErr: newMultiError([]string{
				`index 0: timestamp: is empty`,
			}),
		},
		{
			name:     "identical home and away team ids must produce the expected error",
			testFile: "matches_rows_with_identical_home_away_team_ids.csv",
			wantErr: newMultiError([]string{
				`index 0: home team id and away team id are identical: PTFC`,
			}),
		},
		{
			name:     "winning team id is not home or away team id must produce the expected error",
			testFile: "matches_rows_with_mismatch_winning_team_id.csv",
			wantErr: newMultiError([]string{
				`index 0: winning team id ABC must match either home or away team id`,
			}),
		},
		{
			name:     "duplicate match id must produce the expected error",
			testFile: "matches_rows_with_duplicate_id.csv",
			wantErr: newMultiError([]string{
				`index 1: id 'A1': is duplicate`,
			}),
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

			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantMatches, gotMatches)
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

func newMultiError(messages []string) error {
	mErr := domain.NewMultiError()

	for _, msg := range messages {
		mErr.Add(errors.New(msg))
	}

	return mErr
}
