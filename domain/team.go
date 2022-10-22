package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

var (
	defaultFileSystem fs.FS // TODO: load files via go:embed
)

type Team struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageURL"`
}

type TeamCollection []*Team

func (tc TeamCollection) GetByID(id string) *Team {
	for _, team := range tc {
		if team != nil && team.ID == id {
			return team
		}
	}

	return nil
}

type TeamsJSONLoader struct {
	fSys fs.FS
	path string
}

func (t *TeamsJSONLoader) WithFileSystem(fSys fs.FS) *TeamsJSONLoader {
	t.fSys = fSys
	return t
}

func (t *TeamsJSONLoader) WithPath(path string) *TeamsJSONLoader {
	t.path = path
	return t
}

func (t *TeamsJSONLoader) init() error {
	if t.fSys == nil {
		t.fSys = defaultFileSystem
	}

	if t.path == "" {
		return fmt.Errorf("path: %w", ErrIsEmpty)
	}

	return nil
}

func (t *TeamsJSONLoader) LoadTeams(_ context.Context) (TeamCollection, error) {
	if err := t.init(); err != nil {
		return nil, err
	}

	// read teams config file
	b, err := readFile(t.fSys, t.path)
	if err != nil {
		return nil, err
	}

	// parse file contents
	var content = &struct {
		Teams TeamCollection `json:"teams"`
	}{}
	if err = json.Unmarshal(b, &content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal team collection: %w", err)
	}

	return validateTeams(content.Teams)
}

func readFile(fSys fs.FS, path string) ([]byte, error) {
	f, err := fSys.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file '%s': %w", path, err)
	}

	defer f.Close()

	// read file contents
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read file '%s': %w", path, err)
	}

	return b, nil
}

func validateTeams(teams TeamCollection) (TeamCollection, error) {
	ids := &sync.Map{}

	for idx, team := range teams {
		// validate current team
		if err := validateTeam(team); err != nil {
			return nil, fmt.Errorf("invalid team at index %d: %w", idx, err)
		}

		// check if this team id already exists in the collection
		if _, ok := ids.Load(team.ID); ok {
			return nil, fmt.Errorf("invalid team at index %d: id %s: %w", idx, team.ID, ErrIsDuplicate)
		}
		ids.Store(team.ID, struct{}{})
	}

	return teams, nil
}

func validateTeam(team *Team) error {
	team.ID = strings.Trim(team.ID, " ")
	team.Name = strings.Trim(team.Name, " ")
	team.ImageURL = strings.Trim(team.ImageURL, " ")

	if team.ID == "" {
		return fmt.Errorf("id: %w", ErrIsEmpty)
	}

	if team.Name == "" {
		return fmt.Errorf("name: %w", ErrIsEmpty)
	}

	if team.ImageURL == "" {
		return fmt.Errorf("image url: %w", ErrIsEmpty)
	}

	return nil
}

type teamsAudit struct {
	teams TeamCollection
	mp    *sync.Map
}

func (t *teamsAudit) init() {
	if t.mp == nil {
		t.mp = &sync.Map{}
	}

	for _, team := range t.teams {
		if team == nil {
			continue
		}

		if _, ok := t.mp.Load(team.ID); !ok {
			t.mp.Store(team.ID, 0)
		}
	}
}

func (t *teamsAudit) set(team *Team, val int) bool {
	if team == nil {
		return false
	}

	t.init()
	t.mp.Store(team.ID, val)

	return true
}

func (t *teamsAudit) ack(team *Team) bool {
	if team == nil {
		return false
	}

	t.init()
	val, ok := t.mp.Load(team.ID)
	if !ok {
		return false
	}

	return t.set(team, val.(int)+1)
}

func (t *teamsAudit) validate(mErr MultiError, exactlyOnce bool) {
	t.init()

	var errs []error
	t.mp.Range(func(key, val any) bool {
		if (exactlyOnce && val.(int) != 1) ||
			(!exactlyOnce && val.(int) == 0) {
			errs = append(errs, fmt.Errorf("team id '%s': count %d", key, val))
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
