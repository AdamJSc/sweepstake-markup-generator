package domain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Sweepstake struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ImageURL     string `json:"image_url"`
	Tournament   *Tournament
	Participants ParticipantCollection `json:"participants"`
	Prizes       PrizeSettings         `json:"prizes"`
	Build        bool                  `json:"build"`
}

func (s *Sweepstake) GenerateMarkup() ([]byte, error) {
	// TODO: test this method using actual tournament data to check for regressions
	buf := &bytes.Buffer{}

	// generate outright prize data
	var winner, runnerUp *OutrightPrize
	if s.Prizes.Winner {
		winner = TournamentWinner(s)
	}
	if s.Prizes.RunnerUp {
		runnerUp = TournamentRunnerUp(s)
	}

	// generate ranked prize data
	var mostGoalsConceded, mostYellowCards, quickestOwnGoal, quickestRedCard *RankedPrize
	if s.Prizes.MostGoalsConceded {
		mostGoalsConceded = MostGoalsConceded(s)
	}
	if s.Prizes.MostYellowCards {
		mostYellowCards = MostYellowCards(s)
	}
	if s.Prizes.QuickestOwnGoal {
		quickestOwnGoal = QuickestOwnGoal(s)
	}
	if s.Prizes.QuickestRedCard {
		quickestRedCard = QuickestRedCard(s)
	}

	// set title as sweepstake name, fallback to tournament name if missing
	title := s.Name
	if title == "" {
		title = s.Tournament.Name
	}

	// set image url as sweepstake, fallback to tournament if missing
	imageURL := s.ImageURL
	if imageURL == "" {
		imageURL = s.Tournament.ImageURL
	}

	var lastUpdated string
	if s.Tournament.WithLastUpdated {
		lastUpdated = time.Now().Format("Mon 2 Jan 2006 at 15:04")
	}

	type prizeData struct {
		Winner            *OutrightPrize
		RunnerUp          *OutrightPrize
		MostGoalsConceded *RankedPrize
		MostYellowCards   *RankedPrize
		QuickestOwnGoal   *RankedPrize
		QuickestRedCard   *RankedPrize
	}

	data := struct {
		Title       string
		ImageURL    string
		LastUpdated string
		Prizes      prizeData
		Sweepstake  *Sweepstake
	}{
		Title:       title,
		ImageURL:    imageURL,
		LastUpdated: lastUpdated,
		Prizes: prizeData{
			Winner:            winner,
			RunnerUp:          runnerUp,
			MostGoalsConceded: mostGoalsConceded,
			MostYellowCards:   mostYellowCards,
			QuickestOwnGoal:   quickestOwnGoal,
			QuickestRedCard:   quickestRedCard,
		},
		Sweepstake: s,
	}

	if err := s.Tournament.Template.ExecuteTemplate(buf, "tpl", data); err != nil {
		return nil, fmt.Errorf("cannot execute template: %w", err)
	}

	return buf.Bytes(), nil
}

type Participant struct {
	TeamID string `json:"team_id"`
	Name   string `json:"participant_name"`
}

type ParticipantCollection []*Participant

func (pc ParticipantCollection) GetByTeamID(id string) *Participant {
	for _, participant := range pc {
		if participant != nil && participant.TeamID == id {
			return participant
		}
	}

	return nil
}

type PrizeSettings struct {
	Winner            bool `json:"winner"`
	RunnerUp          bool `json:"runner_up"`
	MostGoalsConceded bool `json:"most_goals_conceded"`
	MostYellowCards   bool `json:"most_yellow_cards"`
	QuickestOwnGoal   bool `json:"quickest_own_goal"`
	QuickestRedCard   bool `json:"quickest_red_card"`
}

type SweepstakeCollection []*Sweepstake

// BytesFunc returns a slice of bytes
type BytesFunc func() ([]byte, error)

// BytesFromFileSystem returns the contents of the file at the provided path within the provided file system
func BytesFromFileSystem(fSys fs.FS, configPath string) BytesFunc {
	return func() ([]byte, error) {
		return readFile(fSys, configPath)
	}
}

type httpDoer interface {
	Do(r *http.Request) (*http.Response, error)
}

// BytesFromURL parses the response body of a GET request to the provided url, using the provided basic auth (optional)
//
// If doer is empty (nil), the net/http package's default client is used
func BytesFromURL(url string, basicAuth string, doer httpDoer) BytesFunc {
	if doer == nil {
		doer = http.DefaultClient
	}

	return func() ([]byte, error) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot create request: %w", err)
		}

		if basicAuth != "" {
			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(basicAuth)))
		}

		resp, err := doer.Do(req)
		if err != nil {
			return nil, fmt.Errorf("cannot perform request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
			return nil, fmt.Errorf("invalid response content type: %s", contentType)
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("cannot read request body: %w", err)
		}

		return b, nil
	}
}

type SweepstakesJSONLoader struct {
	source      BytesFunc
	tournaments TournamentCollection
}

func (s *SweepstakesJSONLoader) WithSource(bytesFn BytesFunc) *SweepstakesJSONLoader {
	s.source = bytesFn
	return s
}

func (s *SweepstakesJSONLoader) WithTournamentCollection(tournaments TournamentCollection) *SweepstakesJSONLoader {
	s.tournaments = tournaments
	return s
}

func (s *SweepstakesJSONLoader) init() error {
	if s.tournaments == nil {
		return fmt.Errorf("tournaments: %w", ErrIsEmpty)
	}

	if s.source == nil {
		return fmt.Errorf("source: %w", ErrIsEmpty)
	}

	return nil
}

func (s *SweepstakesJSONLoader) LoadSweepstakes(_ context.Context) (SweepstakeCollection, error) {
	if err := s.init(); err != nil {
		return nil, err
	}

	// read sweepstake config file
	raw, err := s.source()
	if err != nil {
		return nil, err
	}

	// parse as sweepstakes
	var content = &struct {
		Sweepstakes []struct {
			*Sweepstake
			TournamentID string `json:"tournament_id"`
		} `json:"sweepstakes"`
	}{}
	if err = json.Unmarshal(raw, content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal sweepstakes: %w", err)
	}

	if len(content.Sweepstakes) == 0 {
		return nil, errors.New("no sweepstakes found in source data")
	}

	collection := make(SweepstakeCollection, 0)
	for idx := range content.Sweepstakes {
		sweepstake := content.Sweepstakes[idx].Sweepstake
		tournamentID := content.Sweepstakes[idx].TournamentID

		// inflate tournament
		tournament := s.tournaments.GetByID(tournamentID)
		if tournament == nil {
			return nil, fmt.Errorf("sweepstake index %d: tournament id '%s': %w", idx, tournamentID, ErrNotFound)
		}
		sweepstake.Tournament = tournament

		collection = append(collection, sweepstake)
	}

	return validateSweepstakes(collection)
}

func validateSweepstakes(sweepstakes SweepstakeCollection) (SweepstakeCollection, error) {
	ids := &sync.Map{}
	mErr := NewMultiError()

	for _, sweepstake := range sweepstakes {
		mErrIdx := mErr.WithPrefix(fmt.Sprintf("id '%s'", sweepstake.ID))

		// check if this sweepstake id already exists in the collection
		if _, ok := ids.Load(sweepstake.ID); ok {
			mErrIdx.Add(ErrIsDuplicate)
		}
		ids.Store(sweepstake.ID, struct{}{})

		// run remaining validation
		validateSweepstake(sweepstake, mErr)
	}

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return sweepstakes, nil
}

func validateSweepstake(sweepstake *Sweepstake, mErr MultiError) *Sweepstake {
	sweepstake.ID = strings.Trim(sweepstake.ID, " ")
	sweepstake.Name = strings.Trim(sweepstake.Name, " ")
	sweepstake.ImageURL = strings.Trim(sweepstake.ImageURL, " ")

	if sweepstake.ID == "" {
		mErr.Add(fmt.Errorf("id: %w", ErrIsEmpty))
	}

	if sweepstake.Name == "" {
		mErr.Add(fmt.Errorf("name: %w", ErrIsEmpty))
	}

	if sweepstake.ImageURL == "" {
		mErr.Add(fmt.Errorf("image url: %w", ErrIsEmpty))
	}

	audit := &teamsAudit{teams: sweepstake.Tournament.Teams}
	for idx, participant := range sweepstake.Participants {
		participant.TeamID = strings.Trim(participant.TeamID, " ")
		participant.Name = strings.Trim(participant.Name, " ")

		mErrIdx := mErr.WithPrefix(fmt.Sprintf("participant index %d", idx))

		if ok := audit.ack(&Team{ID: participant.TeamID}); !ok {
			mErrIdx.Add(fmt.Errorf("unrecognised participant team id: %s", participant.TeamID))
		}
	}

	audit.validate(mErr, true)

	return sweepstake
}
