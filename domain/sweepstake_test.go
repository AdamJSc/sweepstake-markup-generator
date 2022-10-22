package domain_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sweepstake.adamjs.net/domain"
)

func TestSweepstakesJSONLoader_LoadSweepstakes(t *testing.T) {
	testTourney1 := &domain.Tournament{
		ID: "TestTourney1",
		Teams: domain.TeamCollection{
			{ID: "BPFC"},
			{ID: "DTFC"},
			{ID: "DYFC"},
			{ID: "HUFC"},
			{ID: "PTFC"},
			{ID: "SJRFC"},
			{ID: "STHFC"},
			{ID: "WTFC"},
		},
	}

	testTourney2 := &domain.Tournament{
		ID: "TestTourney2",
		Teams: domain.TeamCollection{
			{ID: "ABC"},
			{ID: "DEF"},
		},
	}

	defaultTestTournaments := domain.TournamentCollection{
		testTourney1,
		testTourney2,
	}

	tt := []struct {
		name            string
		tournaments     domain.TournamentCollection
		configFilename  string
		wantSweepstakes domain.SweepstakeCollection
		wantErr         error
	}{
		{
			name:           "valid sweepstake json must be loaded successfully",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_ok.json",
			wantSweepstakes: domain.SweepstakeCollection{
				{
					ID:         "test-sweepstake-1",
					Name:       "Test Sweepstake 1",
					ImageURL:   "http://sweepstake1.jpg",
					Tournament: testTourney1,
					Participants: []*domain.Participant{
						{TeamID: "BPFC", Name: "John L"},
						{TeamID: "DTFC", Name: "Paul M"},
						{TeamID: "DYFC", Name: "George H"},
						{TeamID: "HUFC", Name: "Ringo S"},
						{TeamID: "PTFC", Name: "Jon L"},
						{TeamID: "SJRFC", Name: "Steve J"},
						{TeamID: "STHFC", Name: "Paul C"},
						{TeamID: "WTFC", Name: "Sid V / Glen M"},
					},
					Build:           true,
					WithLastUpdated: true,
				},
				{
					ID:         "test-sweepstake-2",
					Name:       "Test Sweepstake 2",
					ImageURL:   "http://sweepstake2.jpg",
					Tournament: testTourney2,
					Participants: []*domain.Participant{
						{TeamID: "ABC", Name: "Dara"},
						{TeamID: "DEF", Name: "Ed"},
					},
					Build: true,
				},
			},
		},
		{
			name:    "empty config filename must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// tournaments are empty
		},
		{
			name:        "empty config filename must produce the expected error",
			tournaments: defaultTestTournaments,
			wantErr:     domain.ErrIsEmpty,
			// configFilename is empty
		},
		{
			name:           "non-existent config file must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "non-existent.json",
			wantErr:        fs.ErrNotExist,
		},
		{
			name:           "invalid sweepstake format must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_unmarshalable.json",
			wantErr: fmt.Errorf("cannot unmarshal sweepstakes: %w", &json.UnmarshalTypeError{
				Value: "number",
				Type:  reflect.TypeOf("string"),
				Field: "sweepstakes.id",
			}),
		},
		{
			name:           "non-existent tournament id must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_non_existent_tournament_id.json",
			wantErr:        domain.ErrNotFound,
		},
		{
			name:           "invalid sweepstake must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_invalid.json",
			wantErr: newMultiError([]string{
				"id: is empty",
				"name: is empty",
				"image url: is empty",
				"participant index 0: unrecognised participant team id: NOT_BPFC",
				"team id 'BPFC': count 0",
				"team id 'WTFC': count 2",
			}),
		},
		{
			name:           "sweepstakes with duplicate id must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_duplicate_id.json",
			wantErr: newMultiError([]string{
				"id 'test-sweepstake-1': is duplicate",
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var configPath string
			if tc.configFilename != "" {
				configPath = filepath.Join(testdataDir, sweepstakesDir, tc.configFilename)
			}

			loader := (&domain.SweepstakesJSONLoader{}).
				WithFileSystem(testdataFilesystem).
				WithTournamentCollection(tc.tournaments).
				WithConfigPath(configPath)

			gotSweepstakes, gotErr := loader.LoadSweepstakes(ctx)
			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantSweepstakes, gotSweepstakes)
		})
	}
}

func parseTemplate(t *testing.T, raw string) *template.Template {
	t.Helper()

	tpl, err := template.New("tpl").Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	return tpl
}

var templateComparer = cmp.Comparer(func(want, got *template.Template) bool {
	// want and got are equal templates if output of execution is the same
	switch {
	case want == nil && got == nil:
		return true
	case want == nil && got != nil:
		return false
	case want != nil && got == nil:
		return false
	}
	wantBuf, gotBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	if err := want.Execute(wantBuf, nil); err != nil {
		return false
	}
	if err := got.Execute(gotBuf, nil); err != nil {
		return false
	}
	return wantBuf.String() == gotBuf.String()
})
