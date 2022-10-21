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

func TestSweepstakeJSONLoader_LoadSweepstake(t *testing.T) {
	testTourney := &domain.Tournament{
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

	defaultTestTournaments := domain.TournamentCollection{
		testTourney,
	}

	tt := []struct {
		name           string
		tournaments    domain.TournamentCollection
		configFilename string
		markupFilename string
		wantSweepstake *domain.Sweepstake
		wantErr        error
	}{
		{
			name:           "valid sweepstake json must be loaded successfully",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstake_config_ok.json",
			markupFilename: "sweepstake_markup_ok.gohtml",
			wantSweepstake: &domain.Sweepstake{
				ID:         "test-sweepstake-1",
				Name:       "Test Sweepstake 1",
				ImageURL:   "http://sweepstake1.jpg",
				Tournament: testTourney,
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
				Template:        parseTemplate(t, "<h1>Hello World</h1>"),
				Build:           true,
				WithLastUpdated: true,
			},
		},
		{
			name:    "empty config filename must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// tournaments are empty
		},
		{
			name:           "empty config filename must produce the expected error",
			tournaments:    defaultTestTournaments,
			markupFilename: "hello world",
			wantErr:        domain.ErrIsEmpty,
			// configFilename is empty
		},
		{
			name:           "empty markup filename must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "hello world",
			wantErr:        domain.ErrIsEmpty,
			// markupFilename is empty
		},
		{
			name:           "non-existent config file must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "non-existent.json",
			markupFilename: "sweepstake_markup_ok.gohtml",
			wantErr:        fs.ErrNotExist,
		},
		{
			name:           "invalid sweepstake format must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstake_config_unmarshalable.json",
			markupFilename: "sweepstake_markup_ok.gohtml",
			wantErr: fmt.Errorf("cannot unmarshal sweepstake: %w", &json.UnmarshalTypeError{
				Value:  "number",
				Struct: "Sweepstake",
				Type:   reflect.TypeOf("string"),
				Field:  "id",
			}),
		},
		{
			name:           "non-existent tournament id must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstake_config_non_existent_tournament_id.json",
			markupFilename: "sweepstake_markup_ok.gohtml",
			wantErr:        domain.ErrNotFound,
		},
		{
			name:           "non-existent markup file must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstake_config_ok.json",
			markupFilename: "non-existent.gohtml",
			wantErr:        fs.ErrNotExist,
		},
		{
			name:           "invalid sweepstake must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstake_config_invalid.json",
			markupFilename: "sweepstake_markup_ok.gohtml",
			wantErr: newMultiError([]string{
				"id: is empty",
				"name: is empty",
				"image url: is empty",
				"participant index 0: unrecognised participant team id: NOT_BPFC",
				"tournament team id 'BPFC', count = 0",
				"tournament team id 'WTFC', count = 2",
			}),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			var configPath, markupPath string
			if tc.configFilename != "" {
				configPath = filepath.Join(testdataDir, sweepstakesDir, tc.configFilename)
			}
			if tc.markupFilename != "" {
				markupPath = filepath.Join(testdataDir, sweepstakesDir, tc.markupFilename)
			}

			loader := (&domain.SweepstakeJSONLoader{}).
				WithFileSystem(testdataFilesystem).
				WithTournamentCollection(tc.tournaments).
				WithConfigPath(configPath).
				WithMarkupPath(markupPath)

			gotSweepstake, gotErr := loader.LoadSweepstake(ctx)
			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantSweepstake, gotSweepstake)
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
	wantBuf, gotBuf := bytes.NewBuffer(nil), bytes.NewBuffer(nil)
	if err := want.Execute(wantBuf, nil); err != nil {
		return false
	}
	if err := got.Execute(gotBuf, nil); err != nil {
		return false
	}
	return wantBuf.String() == gotBuf.String()
})
