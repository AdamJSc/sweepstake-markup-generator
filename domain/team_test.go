package domain_test

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sweepstake.adamjs.net/domain"
)

const (
	matchesDir     = "matches"
	teamsDir       = "teams"
	testdataDir    = "testdata"
	tournamentsDir = "tournaments"
)

var (
	//go:embed testdata
	testdataFilesystem embed.FS
)

func TestTeamCollection_GetByID(t *testing.T) {
	teamA1 := &domain.Team{
		ID:       "teamA",
		Name:     "TeamA",
		ImageURL: "http://team-a1.jpg",
	}

	teamB := &domain.Team{
		ID:       "teamB",
		Name:     "TeamB",
		ImageURL: "http://team-b.jpg",
	}

	teamA2 := &domain.Team{
		ID:       "teamA",
		Name:     "TeamA2",
		ImageURL: "http://team-a2.jpg",
	}

	collection := domain.TeamCollection{
		teamA1,
		teamB,
		teamA2, // duplicate id, should never be returned (teamA1 should match first)
	}

	tt := []struct {
		name     string
		id       string
		wantTeam *domain.Team
	}{
		{
			name:     "duplicate team id must return first matched item",
			id:       "teamA",
			wantTeam: teamA1,
		},
		{
			name:     "unique team id must return only matching item",
			id:       "teamB",
			wantTeam: teamB,
		},
		{
			name: "non-matching item must return nil",
			id:   "teamC",
			// want nil team
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotTeam := collection.GetByID(tc.id)
			cmpDiff(t, tc.wantTeam, gotTeam)
		})
	}
}

func TestTeamsJSONLoader_LoadTeams(t *testing.T) {
	tt := []struct {
		name      string
		testFile  string
		wantTeams domain.TeamCollection
		wantErr   error
	}{
		{
			name:     "valid teams json must be loaded successfully",
			testFile: "teams_ok.json",
			wantTeams: domain.TeamCollection{
				{ID: "BPFC", Name: "Bournemouth Poppies", ImageURL: "http://bpfc.jpg"},
				{ID: "DTFC", Name: "Dorchester Town", ImageURL: "http://dtfc.jpg"},
				{ID: "DYFC", Name: "Dexters Youth", ImageURL: "http://dyfc.jpg"},
				{ID: "HUFC", Name: "Hamworthy United", ImageURL: "http://hufc.jpg"},
				{ID: "PTFC", Name: "Poole Town", ImageURL: "http://ptfc.jpg"},
				{ID: "SJRFC", Name: "St John's Rangers", ImageURL: "http://sjrfc.jpg"},
				{ID: "STHFC", Name: "Swanage Town & Herston", ImageURL: "http://sthfc.jpg"},
				{ID: "WTFC", Name: "Wimborne Town", ImageURL: "http://wtfc.jpg"},
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
			name:     "invalid teams format must produce the expected error",
			testFile: "teams_unmarshalable.json",
			wantErr: fmt.Errorf("cannot unmarshal team collection: %w", &json.UnmarshalTypeError{
				Value: "number",
				Type:  reflect.TypeOf(domain.TeamCollection{}),
				Field: "teams",
			}),
		},
		{
			name:     "empty team id must produce the expected error",
			testFile: "teams_empty_id.json",
			wantErr:  errors.New("invalid team at index 0: id: is empty"),
		},
		{
			name:     "empty team name must produce the expected error",
			testFile: "teams_empty_name.json",
			wantErr:  errors.New("invalid team at index 0: name: is empty"),
		},
		{
			name:     "empty team image url must produce the expected error",
			testFile: "teams_empty_image_url.json",
			wantErr:  errors.New("invalid team at index 0: image url: is empty"),
		},
		{
			name:     "duplicate team id must produce the expected error",
			testFile: "teams_duplicate_id.json",
			wantErr:  fmt.Errorf("invalid team at index 2: id PTFC: %w", domain.ErrIsDuplicate),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var testPath []string
			if tc.testFile != "" {
				testPath = []string{teamsDir, tc.testFile}
			}

			loader := newTeamsJSONLoader(testPath...)
			gotTeams, gotErr := loader.LoadTeams(nil)

			cmpDiff(t, tc.wantTeams, gotTeams)
			cmpError(t, tc.wantErr, gotErr)
		})
	}
}

func newTeamsJSONLoader(path ...string) *domain.TeamsJSONLoader {
	if len(path) > 0 {
		path = append([]string{testdataDir}, path...)
	}
	fullPath := filepath.Join(path...)
	return (&domain.TeamsJSONLoader{}).WithFileSystem(testdataFilesystem).WithPath(fullPath)
}

func cmpDiff(t *testing.T, want, got interface{}) {
	t.Helper()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("mismatch (-want, +got): %s", diff)
	}
}

func cmpError(t *testing.T, wantErr, gotErr error) {
	t.Helper()

	switch {
	case wantErr == nil && gotErr == nil:
		return
	case wantErr == nil && gotErr != nil:
		t.Fatalf("want nil error, got '%s' (%T)", gotErr, gotErr)
	case wantErr != nil && gotErr == nil:
		t.Fatalf("want error '%s' (%T), got nil", wantErr, wantErr)
	case wantErr.Error() != gotErr.Error() && !errors.Is(gotErr, wantErr):
		doFatalMsg := func(diff string) {
			t.Helper()
			t.Fatalf("want error '%s' (%T), got '%s' (%T), diff: %s", wantErr, wantErr, gotErr, gotErr, diff)
		}

		defer func() {
			if r := recover(); r != any(nil) {
				// unable to diff the error objects so just diff the error messages instead
				doFatalMsg(cmp.Diff(wantErr.Error(), gotErr.Error()))
			}
		}()

		doFatalMsg(cmp.Diff(wantErr, gotErr))
	}
}
