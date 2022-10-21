package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"strings"
)

type Sweepstake struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImageURL        string `json:"imageURL"`
	Tournament      *Tournament
	Participants    []Participant `json:"participants"`
	Markup          template.HTML
	Build           bool `json:"build"`
	WithLastUpdated bool `json:"with_last_updated"`
}

type Participant struct {
	TeamID          string `json:"team_id"`
	ParticipantName string `json:"participant_name"`
}

type SweepstakeJSONLoader struct {
	fSys        fs.FS
	tournaments TournamentCollection
	configPath  string
	markupPath  string
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

func (s *SweepstakeJSONLoader) WithMarkupPath(path string) *SweepstakeJSONLoader {
	s.markupPath = path
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

	if s.markupPath == "" {
		return fmt.Errorf("markup path: %w", ErrIsEmpty)
	}

	return nil
}

func (s *SweepstakeJSONLoader) LoadSweepstake(_ context.Context) (*Sweepstake, error) {
	if err := s.init(); err != nil {
		return nil, err
	}

	// read sweepstake config file
	b, err := readFile(s.fSys, s.configPath)
	if err != nil {
		return nil, err
	}

	// parse as sweepstake
	sweepstake := &Sweepstake{}
	if err = json.Unmarshal(b, sweepstake); err != nil {
		return nil, fmt.Errorf("cannot unmarshal sweepstake: %w", err)
	}

	// inflate tournament
	var content = &struct {
		TournamentID string `json:"tournament_id"`
	}{}
	if err = json.Unmarshal(b, &content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal tournament id: %w", err)
	}

	tournament := s.tournaments.GetByID(strings.Trim(content.TournamentID, " "))
	if tournament == nil {
		return nil, fmt.Errorf("tournament id '%s': %w", content.TournamentID, ErrNotFound)
	}

	sweepstake.Tournament = tournament

	// TODO: read and parse markup file

	return validateSweepstake(sweepstake)
}

func validateSweepstake(sweepstake *Sweepstake) (*Sweepstake, error) {
	sweepstake.ID = strings.Trim(sweepstake.ID, " ")
	sweepstake.Name = strings.Trim(sweepstake.Name, " ")
	sweepstake.ImageURL = strings.Trim(sweepstake.ImageURL, " ")

	if sweepstake.ID == "" {
		return nil, fmt.Errorf("id: %w", ErrIsEmpty)
	}

	if sweepstake.Name == "" {
		return nil, fmt.Errorf("name: %w", ErrIsEmpty)
	}

	if sweepstake.ImageURL == "" {
		return nil, fmt.Errorf("image url: %w", ErrIsEmpty)
	}

	// TODO: add extra sweepstake validation rules

	return sweepstake, nil
}
