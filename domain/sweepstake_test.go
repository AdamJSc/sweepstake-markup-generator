package domain_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sweepstake-markup-generator/domain"
)

func TestBytesFromFileSystem(t *testing.T) {
	tt := []struct {
		name       string
		fileSystem fs.FS
		configPath string
		wantBytes  []byte
		wantErr    error
	}{
		{
			name:       "existent file must return the expected bytes",
			fileSystem: testdataFilesystem,
			configPath: "sweepstakes_ok.json",
			wantBytes:  readTestDataFile(t, sweepstakesDir, "sweepstakes_ok.json"),
			// want no error
		},
		{
			name:       "non-existent file must return the expected error",
			fileSystem: testdataFilesystem,
			configPath: "non-existent.json",
			wantErr:    fs.ErrNotExist,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.configPath
			if path != "" {
				path = filepath.Join(testdataDir, sweepstakesDir, path)
			}

			gotBytes, gotErr := domain.BytesFromFileSystem(tc.fileSystem, path)()
			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantBytes, gotBytes)
		})
	}
}

type doFunc func(r *http.Request) (*http.Response, error)

func (d doFunc) Do(r *http.Request) (*http.Response, error) {
	return d(r)
}

func okResponse() *http.Response {
	header := http.Header{}
	header.Set("Content-Type", "application/json")

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(bytes.NewReader([]byte(`hello world`))),
	}
}

type errReader struct{ err error }

func (e errReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestBytesFromURL(t *testing.T) {
	tt := []struct {
		name      string
		url       string
		basicAuth string
		doFunc    doFunc
		wantBytes []byte
		wantErr   error
	}{
		{
			name:      "successful http response must return the expected bytes",
			url:       "http://my-url",
			basicAuth: "hello:world",
			doFunc: doFunc(func(r *http.Request) (*http.Response, error) {
				wantURL := "http://my-url"
				wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("hello:world"))
				if gotURL := r.URL.String(); gotURL != wantURL {
					return nil, fmt.Errorf("want url '%s', got '%s'", wantURL, gotURL)
				}
				if gotAuth := r.Header.Get("Authorization"); gotAuth != wantAuth {
					return nil, fmt.Errorf("want basic auth '%s', got '%s'", wantAuth, gotAuth)
				}
				return okResponse(), nil
			}),
			wantBytes: []byte(`hello world`),
			// want no error
		},
		{
			name: "failure to perform request must produce the expected error",
			doFunc: doFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("oops")
			}),
			wantErr: errors.New("cannot perform request: oops"),
		},
		{
			name: "invalid response status code must produce the expected error",
			doFunc: doFunc(func(r *http.Request) (*http.Response, error) {
				resp := okResponse()
				// set status code to invalid value
				resp.StatusCode = 123
				return resp, nil
			}),
			wantErr: errors.New("non-200 status code: 123"),
		},
		{
			name: "invalid response content type must produce the expected error",
			doFunc: doFunc(func(r *http.Request) (*http.Response, error) {
				resp := okResponse()
				// override content-type header value
				resp.Header.Set("Content-Type", "lololol")
				return resp, nil
			}),
			wantErr: errors.New("invalid response content type: lololol"),
		},
		{
			name: "response body that returns error on read must produce the expected error",
			doFunc: doFunc(func(r *http.Request) (*http.Response, error) {
				resp := okResponse()
				// body returns read error
				resp.Body = io.NopCloser(errReader{err: errors.New("oops")})
				return resp, nil
			}),
			wantErr: errors.New("cannot read request body: oops"),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotBytes, gotErr := domain.BytesFromURL(tc.url, tc.basicAuth, tc.doFunc)()
			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantBytes, gotBytes)
		})
	}
}

func TestParticipantCollection_GetByTeamID(t *testing.T) {
	participantA1 := &domain.Participant{
		TeamID: "teamA",
	}

	participantB := &domain.Participant{
		TeamID: "teamB",
	}

	participantA2 := &domain.Participant{
		TeamID: "teamA",
	}

	collection := domain.ParticipantCollection{
		participantA1,
		participantB,
		participantA2, // duplicate id, should never be returned (participantA1 should match first)
	}

	tt := []struct {
		name            string
		id              string
		wantParticipant *domain.Participant
	}{
		{
			name:            "duplicate participant id must return first matched item",
			id:              "teamA",
			wantParticipant: participantA1,
		},
		{
			name:            "unique participant id must return only matching item",
			id:              "teamB",
			wantParticipant: participantB,
		},
		{
			name: "non-matching item must return nil",
			id:   "teamC",
			// want nil participant
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotMatch := collection.GetByTeamID(tc.id)
			cmpDiff(t, tc.wantParticipant, gotMatch)
		})
	}
}

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
					Prizes: domain.PrizeSettings{
						Winner:            true,
						RunnerUp:          true,
						MostGoalsConceded: true,
						MostYellowCards:   true,
						QuickestOwnGoal:   true,
						QuickestRedCard:   true,
					},
					Build: true,
				},
				{
					ID:         "test-sweepstake-2",
					Name:       "Test Sweepstake 2",
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
			name:    "empty tournaments must produce the expected error",
			wantErr: domain.ErrIsEmpty,
			// tournaments are empty
		},
		{
			name:        "empty config filename must produce the expected error",
			tournaments: defaultTestTournaments,
			wantErr:     errors.New("cannot open file '': open : file does not exist"),
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
			name:           "no sweepstakes must produce the expected error",
			tournaments:    defaultTestTournaments,
			configFilename: "sweepstakes_none.json",
			wantErr:        errors.New("no sweepstakes found in source data"),
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

			loader := newSweepstakesJSONLoader(tc.configFilename).
				WithTournamentCollection(tc.tournaments)

			gotSweepstakes, gotErr := loader.LoadSweepstakes(ctx)
			cmpError(t, tc.wantErr, gotErr)
			cmpDiff(t, tc.wantSweepstakes, gotSweepstakes)
		})
	}
}

func newSweepstakesJSONLoader(path string) *domain.SweepstakesJSONLoader {
	if path != "" {
		path = filepath.Join(testdataDir, sweepstakesDir, path)
	}

	return (&domain.SweepstakesJSONLoader{}).
		WithSource(domain.BytesFromFileSystem(testdataFilesystem, path))
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

func readTestDataFile(t *testing.T, path ...string) []byte {
	t.Helper()
	path = append([]string{"testdata"}, path...)

	b, err := os.ReadFile(filepath.Join(path...))
	if err != nil {
		t.Fatal(err)
	}

	return b
}
