package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
)

var (
	defaultFileSystem fs.FS // TODO: load files via go:embed
	ErrIsEmpty        = errors.New("is empty")
)

type Team struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageURL"`
}

type TeamCollection []Team

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

	// open file
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

	return content.Teams, nil
}
