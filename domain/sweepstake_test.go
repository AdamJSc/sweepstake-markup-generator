package domain_test

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sweepstake.adamjs.net/domain"
)

func TestSweepstakeJSONLoader_LoadSweepstake(t *testing.T) {
	defaultTestTournaments := domain.TournamentCollection{
		{
			ID: "TestTourney1",
		},
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
				Tournament: &domain.Tournament{ID: "TestTourney1"},
				Participants: []domain.Participant{
					{TeamID: "BPFC ", ParticipantName: "John L "},
					{TeamID: "DTFC", ParticipantName: "Paul M"},
					{TeamID: "DYFC", ParticipantName: "George H"},
					{TeamID: "HUFC", ParticipantName: "Ringo S"},
					{TeamID: "PTFC", ParticipantName: "Jon L"},
					{TeamID: "SJRFC", ParticipantName: "Steve J"},
					{TeamID: "STHFC", ParticipantName: "Paul C"},
					{TeamID: "WTFC", ParticipantName: "Sid V / Glen M"},
				},
				Markup:          template.HTML(""),
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
		// TODO: add more sad path tests
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
