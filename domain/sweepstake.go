package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
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

type SweepstakeJSONLoader struct {
	fSys        fs.FS
	tournaments TournamentCollection
	configPath  string
}

func (s *SweepstakeJSONLoader) WithFileSystem(fSys fs.FS) *SweepstakeJSONLoader {
	s.fSys = fSys
	return s
}

func (s *SweepstakeJSONLoader) WithTournamentCollection(tournaments TournamentCollection) *SweepstakeJSONLoader {
	s.tournaments = tournaments
	return s
}

func (s *SweepstakeJSONLoader) WithConfigPath(path string) *SweepstakeJSONLoader {
	s.configPath = path
	return s
}

func (s *SweepstakeJSONLoader) init() error {
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

func (s *SweepstakeJSONLoader) LoadSweepstake(_ context.Context) (*Sweepstake, error) {
	if err := s.init(); err != nil {
		return nil, err
	}

	// read sweepstake config file
	rawConfigJSON, err := readFile(s.fSys, s.configPath)
	if err != nil {
		return nil, err
	}

	// parse as sweepstake
	sweepstake := &Sweepstake{}
	if err = json.Unmarshal(rawConfigJSON, sweepstake); err != nil {
		return nil, fmt.Errorf("cannot unmarshal sweepstake: %w", err)
	}

	// inflate tournament
	var content = &struct {
		TournamentID string `json:"tournament_id"`
	}{}
	if err = json.Unmarshal(rawConfigJSON, &content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal tournament id: %w", err)
	}

	tournament := s.tournaments.GetByID(content.TournamentID)
	if tournament == nil {
		return nil, fmt.Errorf("tournament id '%s': %w", content.TournamentID, ErrNotFound)
	}

	sweepstake.Tournament = tournament

	return validateSweepstake(sweepstake)
}

func validateSweepstake(sweepstake *Sweepstake) (*Sweepstake, error) {
	mErr := NewMultiError()

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

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return sweepstake, nil
}
