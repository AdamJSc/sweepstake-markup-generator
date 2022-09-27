package domain

type Match struct {
	Home MatchCompetitor
	Away MatchCompetitor
}

type MatchCompetitor struct {
	Team   *Team
	Events []MatchEvent
}

type MatchEvent struct {
	EventType MatchEventType
	Name      string
	Minute    uint16
}

type MatchEventType uint8

const (
	// TODO: add more match event types and convert to string values
	_       MatchEventType = iota
	OwnGoal MatchEventType = iota
	YellowCard
)
