package domain

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"strings"
	"sync"
	"time"
)

type Match struct {
	ID        string
	Date      time.Time
	Stage     MatchStage
	Home      MatchCompetitor
	Away      MatchCompetitor
	Winner    *Team
	Completed bool
}

type MatchStage uint8

const (
	// TODO: add more match stages and convert to string values
	_ MatchStage = iota
	Group
	Knockout
)

type MatchCompetitor struct {
	Team        *Team
	Goals       uint8
	YellowCards uint8
	OwnGoals    []MatchEvent
	RedCards    []MatchEvent
}

type MatchEvent struct {
	Name   string
	Minute float32
}

type MatchCollection []*Match

type MatchesCSVLoader struct {
	fSys fs.FS
	path string
}

func (m *MatchesCSVLoader) WithFileSystem(fSys fs.FS) *MatchesCSVLoader {
	m.fSys = fSys
	return m
}

func (m *MatchesCSVLoader) WithPath(path string) *MatchesCSVLoader {
	m.path = path
	return m
}

func (m *MatchesCSVLoader) init() error {
	if m.fSys == nil {
		m.fSys = defaultFileSystem
	}

	if m.path == "" {
		return fmt.Errorf("path: %w", ErrIsEmpty)
	}

	return nil
}

func (m *MatchesCSVLoader) LoadMatches(_ context.Context) (MatchCollection, error) {
	if err := m.init(); err != nil {
		return nil, err
	}

	// open file
	f, err := m.fSys.Open(m.path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	defer f.Close()

	// read file contents
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	// TODO: parse file contents as csv
	// TODO: transform file structure to matches collection
	var matches MatchCollection
	log.Println(string(b))

	return validateMatches(matches)
}

func validateMatches(matches MatchCollection) (MatchCollection, error) {
	ids := &sync.Map{}

	for idx, match := range matches {
		// validate current match
		if err := validateMatch(match); err != nil {
			return nil, fmt.Errorf("invalid match at index %d: %w", idx, err)
		}

		// check if this match id already exists in the collection
		if _, ok := ids.Load(match.ID); ok {
			return nil, fmt.Errorf("invalid match at index %d: id %s: %w", idx, match.ID, ErrIsDuplicate)
		}
		ids.Store(match.ID, struct{}{})
	}

	return matches, nil
}

func validateMatch(match *Match) error {
	// TODO: add match sanitisation and validation rules
	match.ID = strings.Trim(match.ID, " ")

	if match.ID == "" {
		return fmt.Errorf("id: %w", ErrIsEmpty)
	}

	return nil
}
