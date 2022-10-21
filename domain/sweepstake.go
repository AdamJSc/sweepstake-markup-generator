package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

type Sweepstake struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ImageURL        string `json:"imageURL"`
	Tournament      *Tournament
	Participants    []*Participant `json:"participants"`
	Template        *template.Template
	Build           bool `json:"build"`
	WithLastUpdated bool `json:"with_last_updated"`
}

type Participant struct {
	TeamID string `json:"team_id"`
	Name   string `json:"participant_name"`
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

	// parse markup as template
	rawMarkup, err := readFile(s.fSys, s.markupPath)
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("tpl").Parse(string(rawMarkup))
	if err != nil {
		return nil, fmt.Errorf("cannot parse template: %w", err)
	}

	sweepstake.Template = tpl

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

	teamsMap := newTeamsIDMap(sweepstake.Tournament.Teams, "tournament team")
	for idx, participant := range sweepstake.Participants {
		participant.TeamID = strings.Trim(participant.TeamID, " ")
		participant.Name = strings.Trim(participant.Name, " ")

		mErrIdx := mErr.WithPrefix(fmt.Sprintf("participant index %d", idx))

		val, ok := teamsMap.load(participant.TeamID)
		if !ok {
			mErrIdx.Add(fmt.Errorf("unrecognised participant team id: %s", participant.TeamID))
		}

		teamsMap.store(participant.TeamID, val+1)
	}

	teamsMap.validate(mErr)

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return sweepstake, nil
}

type idMap struct {
	*sync.Map
	name string
}

func (i *idMap) init() {
	if i.Map == nil {
		i.Map = &sync.Map{}
	}
}

func (i *idMap) store(key string, val int) {
	i.init()
	i.Map.Store(key, val)
}

func (i *idMap) load(key string) (int, bool) {
	i.init()
	val, ok := i.Map.Load(key)
	if !ok {
		return 0, false
	}
	return val.(int), ok
}

func (i *idMap) validate(mErr MultiError) {
	prefix := "id"
	if i.name != "" {
		prefix = i.name + " id"
	}

	var errs []error
	i.Map.Range(func(key, val any) bool {
		if val.(int) != 1 {
			errs = append(errs, fmt.Errorf("%s '%s', count = %d", prefix, key, val))
		}
		return true
	})

	// guarantee error order
	sort.SliceStable(errs, func(i, j int) bool {
		return errs[i].Error() < errs[j].Error()
	})

	for _, err := range errs {
		mErr.Add(err)
	}
}

func newTeamsIDMap(teams TeamCollection, name string) *idMap {
	mp := &idMap{name: name}

	for _, team := range teams {
		mp.store(team.ID, 0)
	}

	return mp
}
