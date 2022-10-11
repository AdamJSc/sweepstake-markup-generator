package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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

	// open teams config file
	f, err := t.fSys.Open(t.path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	defer f.Close()

	// read file contents
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
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
