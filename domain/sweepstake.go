package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type Sweepstake struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImageURL        string `json:"imageURL"`
	Tournament      *Tournament
	Participants    []*Participant `json:"participants"`
	Build           bool           `json:"build"`
	WithLastUpdated bool           `json:"with_last_updated"`
}

type Participant struct {
	TeamID string `json:"team_id"`
	Name   string `json:"participant_name"`
}

type SweepstakeCollection []*Sweepstake

type SweepstakesJSONLoader struct {
	fSys        fs.FS
	tournaments TournamentCollection
	configPath  string
}

func (s *SweepstakesJSONLoader) WithFileSystem(fSys fs.FS) *SweepstakesJSONLoader {
	s.fSys = fSys
	return s
}

func (s *SweepstakesJSONLoader) WithTournamentCollection(tournaments TournamentCollection) *SweepstakesJSONLoader {
	s.tournaments = tournaments
	return s
}

func (s *SweepstakesJSONLoader) WithConfigPath(path string) *SweepstakesJSONLoader {
	s.configPath = path
	return s
}

func (s *SweepstakesJSONLoader) init() error {
	if s.fSys == nil {
		s.fSys = defaultFileSystem
	}

	if s.tournaments == nil {
		return fmt.Errorf("tournaments: %w", ErrIsEmpty)
	}

	if s.configPath == "" {
		return fmt.Errorf("config path: %w", ErrIsEmpty)
	}

	return nil
}

func (s *SweepstakesJSONLoader) LoadSweepstakes(_ context.Context) (SweepstakeCollection, error) {
	if err := s.init(); err != nil {
		return nil, err
	}

	// read sweepstake config file
	rawConfigJSON, err := readFile(s.fSys, s.configPath)
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
	if err = json.Unmarshal(rawConfigJSON, content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal sweepstakes: %w", err)
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
