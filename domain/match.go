package domain

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-cmp/cmp"
)

type Match struct {
	ID        string
	Timestamp time.Time
	Stage     MatchStage
	Home      MatchCompetitor
	Away      MatchCompetitor
	Winner    *Team
	Completed bool
}

type MatchStage uint8

const (
	_ MatchStage = iota
	GroupStage
	KnockoutStage
)

var matchesCSVHeader = []string{
	"MATCH_ID",
	"DATE",
	"TIME",
	"STAGE",
	"COMPLETED",
	"WINNER_TEAM_ID",
	"HOME_TEAM_ID",
	"AWAY_TEAM_ID",
	"HOME_GOALS",
	"AWAY_GOALS",
	"HOME_YELLOW_CARDS",
	"AWAY_YELLOW_CARDS",
	"HOME_OG",
	"AWAY_OG",
	"HOME_RED_CARDS",
	"AWAY_RED_CARDS",
}

type MatchCompetitor struct {
	Team        *Team
	Goals       uint8
	YellowCards uint8
	OwnGoals    []MatchEvent
	RedCards    []MatchEvent
}

type MatchEvent struct {
	Name   string // name of player who performed the event
	Minute uint8  // match minute event took place
	Offset uint8  // indicates if event took place in stopped time - e.g. 90+2 = offset 2
}

type MatchCollection []*Match

func (mc MatchCollection) GetByID(id string) *Match {
	for _, match := range mc {
		if match != nil && match.ID == id {
			return match
		}
	}

	return nil
}

func (mc MatchCollection) GetWinnerByMatchID(id string) *Team {
	match := mc.GetByID(id)

	if match == nil || !match.Completed {
		return nil
	}

	return match.Winner
}

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

	// open matches csv file
	f, err := m.fSys.Open(m.path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}

	defer f.Close()

	// parse file contents
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("cannot read file: %w", err)
	}

	// transform and validate
	matches, err := transformCSVToMatches(records)
	if err != nil {
		return nil, fmt.Errorf("cannot transform csv: %w", err)
	}

	return validateMatches(matches)
}

func transformCSVToMatches(records [][]string) (MatchCollection, error) {
	if len(records) < 2 {
		return nil, fmt.Errorf("rows %d: file must have header row and at least one more row", len(records))
	}
	headerRow := records[0]
	if diff := cmp.Diff(headerRow, matchesCSVHeader); diff != "" {
		return nil, fmt.Errorf("invalid headers: %s", strings.Join(headerRow, ","))
	}

	var (
		matches MatchCollection
		mErr    = NewMultiError()
	)

	for idx, row := range records[1:] {
		rowNum := idx + 1
		mErrRow := mErr.WithPrefix(fmt.Sprintf("row %d", rowNum))
		match := transformCSVRowToMatch(row, mErrRow)
		matches = append(matches, match)
	}

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return matches, nil
}

func transformCSVRowToMatch(row []string, mErr MultiError) *Match {
	matchID := row[0]             // MATCH_ID
	sDate := row[1]               // DATE
	sTime := row[2]               // TIME
	rawStage := row[3]            // STAGE
	rawCompleted := row[4]        // COMPLETED
	winnerTeamID := row[5]        // WINNER_TEAM_ID
	homeTeamID := row[6]          // HOME_TEAM_ID
	awayTeamID := row[7]          // AWAY_TEAM_ID
	rawHomeGoals := row[8]        // HOME_GOALS
	rawAwayGoals := row[9]        // AWAY_GOALS
	rawHomeYellowCards := row[10] // HOME_YELLOW_CARDS
	rawAwayYellowCards := row[11] // AWAY_YELLOW_CARDS
	rawHomeOG := row[12]          // HOME_OG
	rawAwayOG := row[13]          // AWAY_OG
	rawHomeRedCards := row[14]    // HOME_RED_CARDS
	rawAwayRedCards := row[15]    // AWAY_RED_CARDS

	match := &Match{
		ID:        matchID,
		Timestamp: parseTimestamp(sDate, sTime, mErr),
		Stage:     convertToMatchStage(rawStage, mErr),
		Home: MatchCompetitor{
			Goals:       parseUInt8(rawHomeGoals, mErr.WithPrefix("home goals")),
			YellowCards: parseUInt8(rawHomeYellowCards, mErr.WithPrefix("home yellow cards")),
			OwnGoals:    parseMatchEvents(rawHomeOG, mErr.WithPrefix("home own goals")),
			RedCards:    parseMatchEvents(rawHomeRedCards, mErr.WithPrefix("home red cards")),
		},
		Away: MatchCompetitor{
			Goals:       parseUInt8(rawAwayGoals, mErr.WithPrefix("away goals")),
			YellowCards: parseUInt8(rawAwayYellowCards, mErr.WithPrefix("away yellow cards")),
			OwnGoals:    parseMatchEvents(rawAwayOG, mErr.WithPrefix("away own goals")),
			RedCards:    parseMatchEvents(rawAwayRedCards, mErr.WithPrefix("away red cards")),
		},
		Completed: rawCompleted == "Y",
	}

	if homeTeamID != "" {
		match.Home.Team = &Team{
			ID: homeTeamID, // id is used as a lookup when later inflating within the context of a tournament
		}
	}
	if awayTeamID != "" {
		match.Away.Team = &Team{
			ID: awayTeamID, // id is used as a lookup when later inflating within the context of a tournament
		}
	}
	if winnerTeamID != "" {
		match.Winner = &Team{
			ID: winnerTeamID, // id is used as a lookup when later inflating within the context of a tournament
		}
	}

	return match
}

func parseTimestamp(sDate, sTime string, mErr MultiError) time.Time {
	sTimestamp := strings.Trim(sDate+" "+sTime, " ")
	if sTimestamp == "" {
		return time.Time{}
	}

	timestamp, err := time.Parse("02/01/2006 15:04", sTimestamp)
	if err != nil {
		mErr.Add(fmt.Errorf("invalid timestamp format: %s", sTimestamp))
		return time.Time{}
	}

	return timestamp
}

func parseUInt8(sInt string, mErr MultiError) uint8 {
	if sInt == "" {
		return 0
	}

	val, err := strconv.Atoi(sInt)
	if err != nil {
		mErr.Add(fmt.Errorf("invalid int: %w", err))
		return 0
	}

	return uint8(val)
}

func parseMatchEvents(sEvents string, mErr MultiError) []MatchEvent {
	sEvents = strings.Trim(sEvents, " ")
	if sEvents == "" {
		return nil
	}

	split := strings.Split(sEvents, ";")
	elemCount, err := strconv.Atoi(split[0])
	if err != nil {
		mErr.Add(errors.New("first element must provide count of remaining elements"))
		return nil
	}

	elems := split[1:]
	if len(elems) != elemCount {
		fmtMsg := "must have %d element"
		if elemCount != 1 {
			fmtMsg += "s"
		}
		mErr.Add(fmt.Errorf(fmtMsg, elemCount))
		return nil
	}

	var events []MatchEvent
	for idx, elem := range elems {
		event := parseMatchEvent(elem, mErr.WithPrefix(fmt.Sprintf("event %d", idx+1)))
		if event != nil {
			events = append(events, *event)
		}
	}

	if len(events) > 0 {
		return events
	}

	return nil
}

func parseMatchEvent(sEvent string, mErr MultiError) *MatchEvent {
	split := strings.Split(sEvent, ":")
	if len(split) != 2 {
		mErr.Add(errors.New("invalid format"))
		return nil
	}

	name := strings.Trim(split[0], " ")
	minuteWithOffset := split[1]

	split = strings.SplitN(minuteWithOffset, "+", 2)
	rawMinute := split[0]
	rawOffset := ""
	if len(split) == 2 {
		rawOffset = split[1]
	}

	minute, err := strconv.Atoi(rawMinute)
	if err != nil {
		mErr.Add(fmt.Errorf("minute: invalid int: %w", err))
		return nil
	}

	if minute < 1 {
		mErr.Add(errors.New("minute: must be greater than 0"))
		return nil
	}

	var offset int
	if rawOffset != "" {
		offset, err = strconv.Atoi(rawOffset)
		if err != nil {
			mErr.Add(fmt.Errorf("offset: invalid int: %w", err))
			return nil
		}

		if offset < 1 {
			mErr.Add(errors.New("offset: must be greater than 0"))
			return nil
		}
	}

	return &MatchEvent{
		Name:   name,
		Minute: uint8(minute),
		Offset: uint8(offset),
	}
}

func convertToMatchStage(s string, mErr MultiError) MatchStage {
	switch s {
	case "GROUP":
		return GroupStage
	case "KO":
		return KnockoutStage
	default:
		mErr.Add(fmt.Errorf("invalid match stage: %s", s))
		return 0
	}
}

func validateMatches(matches MatchCollection) (MatchCollection, error) {
	ids := &sync.Map{}
	mErr := NewMultiError()

	for idx, match := range matches {
		// validate current match
		mErrIdx := mErr.WithPrefix(fmt.Sprintf("index %d", idx))
		validateMatch(match, mErrIdx)

		// check if this match id already exists in the collection
		if _, ok := ids.Load(match.ID); ok {
			mErrIdx.Add(fmt.Errorf("id '%s': %w", match.ID, ErrIsDuplicate))
		}
		ids.Store(match.ID, struct{}{})
	}

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return matches, nil
}

func validateMatch(match *Match, mErr MultiError) {
	match.ID = strings.Trim(match.ID, " ")

	if match.Home.Team != nil {
		match.Home.Team.ID = strings.Trim(match.Home.Team.ID, " ")
	}

	if match.Away.Team != nil {
		match.Away.Team.ID = strings.Trim(match.Away.Team.ID, " ")
	}

	if match.Winner != nil {
		match.Winner.ID = strings.Trim(match.Winner.ID, " ")
	}

	if match.ID == "" {
		mErr.Add(fmt.Errorf("id: %w", ErrIsEmpty))
	}

	if match.Timestamp.Equal(time.Time{}) {
		mErr.Add(fmt.Errorf("timestamp: %w", ErrIsEmpty))
	}

	if isTeamIDIdentical(match.Home.Team, match.Away.Team) {
		mErr.Add(fmt.Errorf("home team id and away team id are identical: %s", match.Home.Team.ID))
	}

	if isTeamNotOneOf(match.Winner, match.Home.Team, match.Away.Team) {
		mErr.Add(fmt.Errorf("winning team id %s must match either home or away team id", match.Winner.ID))
	}
}

func isTeamIDIdentical(a, b *Team) bool {
	if a == nil || b == nil {
		return false // return early
	}

	return a.ID == b.ID
}

func isTeamNotOneOf(needle *Team, haystack ...*Team) bool {
	if needle == nil {
		return false // exit early
	}

	for _, t := range haystack {
		if t != nil && t.ID == needle.ID {
			return false // needle is one of haystack
		}
	}

	return true // needle is not one of haystack
}
