package domain

import (
	"fmt"
	"sort"
	"time"
)

const (
	// finalMatchID defines the id of the match considered to be the final
	finalMatchID       = "F"
	mostGoalsConceded  = "Most Goals Conceded"
	mostYellowCards    = "Most Yellow Cards"
	quickestOwnGoal    = "Quickest Own Goal"
	tournamentRunnerUp = "Tournament Runner-Up"
	tournamentWinner   = "Tournament Winner"
)

// OutrightPrize represents a prize with a single outright winner
type OutrightPrize struct {
	PrizeName       string
	ParticipantName string
	ImageURL        string
}

// OutrightPrizeGenerator defines a function that generates an outright prize from the provided Sweepstake
type OutrightPrizeGenerator func(sweepstake *Sweepstake) *OutrightPrize

// TournamentWinner determines the winner of the provided Sweepstake
var TournamentWinner = func(s *Sweepstake) *OutrightPrize {
	defaultPrize := &OutrightPrize{
		PrizeName:       tournamentWinner,
		ParticipantName: "TBC",
	}

	if s == nil {
		return defaultPrize
	}

	// get match winner
	winningTeam := s.Tournament.Matches.GetWinnerByMatchID(finalMatchID)
	if winningTeam == nil {
		return defaultPrize
	}

	// get participant who represents the match winner
	participant := s.Participants.GetByTeamID(winningTeam.ID)
	winnerName := getSummaryFromTeamAndParticipant(winningTeam, participant)

	return &OutrightPrize{
		PrizeName:       tournamentWinner,
		ParticipantName: winnerName,
		ImageURL:        winningTeam.ImageURL,
	}
}

func getSummaryFromTeamAndParticipant(team *Team, participant *Participant) string {
	if participant == nil || participant.Name == "" {
		return team.Name
	}

	return fmt.Sprintf("%s (%s)", participant.Name, team.Name)
}

// TournamentRunnerUp determines the runner-up of the provided Sweepstake
var TournamentRunnerUp = func(s *Sweepstake) *OutrightPrize {
	defaultPrize := &OutrightPrize{
		PrizeName:       tournamentRunnerUp,
		ParticipantName: "TBC",
	}

	if s == nil {
		return defaultPrize
	}

	// get match runner-up
	runnerUpTeam := s.Tournament.Matches.GetRunnerUpByMatchID(finalMatchID)
	if runnerUpTeam == nil {
		return defaultPrize
	}

	// get participant who represents the match runner-up
	participant := s.Participants.GetByTeamID(runnerUpTeam.ID)
	participantSummary := getSummaryFromTeamAndParticipant(runnerUpTeam, participant)

	return &OutrightPrize{
		PrizeName:       tournamentRunnerUp,
		ParticipantName: participantSummary,
		ImageURL:        runnerUpTeam.ImageURL,
	}
}

// MostGoalsConceded returns the teams who have conceded the most goals in descending order
var MostGoalsConceded = func(s *Sweepstake) *RankedPrize {
	defaultPrize := &RankedPrize{
		PrizeName: mostGoalsConceded,
		Rankings:  make([]Rank, 0),
	}

	if s == nil {
		return defaultPrize
	}

	totals := teamsAudit{teams: s.Tournament.Teams}

	for _, match := range s.Tournament.Matches {
		if !match.Completed {
			continue
		}

		totals.inc(match.Home.Team, int(match.Away.Goals)) // goals scored by away team are conceded by home team
		totals.inc(match.Away.Team, int(match.Home.Goals)) // goals scored by home team are conceded by away team
	}

	return &RankedPrize{
		PrizeName: mostGoalsConceded,
		Rankings:  getPrizeRankingsFromAudit(totals, s.Participants),
	}
}

func getPrizeRankingsFromAudit(audit teamsAudit, participants ParticipantCollection) []Rank {
	type teamWithValue struct {
		team  *Team
		value int
	}

	results := make([]teamWithValue, 0)

	for _, t := range audit.teams {
		val, _ := audit.get(t)
		results = append(results, teamWithValue{
			team:  t,
			value: val,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].value > results[j].value
	})

	ranks := make([]Rank, 0)

	for idx, result := range results {
		if result.value == 0 {
			continue
		}

		ranks = append(ranks, Rank{
			Position:        uint8(idx + 1),
			ImageURL:        result.team.ImageURL,
			ParticipantName: getSummaryFromTeamAndParticipant(result.team, participants.GetByTeamID(result.team.ID)),
			Value:           fmt.Sprintf("⚽️ %d", result.value),
		})
	}

	return ranks
}

// MostYellowCards returns the teams who have received the most yellow cards in descending order
var MostYellowCards = func(s *Sweepstake) *RankedPrize {
	defaultPrize := &RankedPrize{
		PrizeName: mostYellowCards,
		Rankings:  make([]Rank, 0),
	}

	if s == nil {
		return defaultPrize
	}

	totals := teamsAudit{teams: s.Tournament.Teams}

	for _, match := range s.Tournament.Matches {
		if !match.Completed {
			continue
		}

		totals.inc(match.Home.Team, int(match.Home.YellowCards))
		totals.inc(match.Away.Team, int(match.Away.YellowCards))
	}

	return &RankedPrize{
		PrizeName: mostYellowCards,
		Rankings:  getPrizeRankingsFromAudit(totals, s.Participants),
	}
}

// QuickestOwnGoal returns the teams who have scored at least one own goal in ascending order of match minute
var QuickestOwnGoal = func(s *Sweepstake) *RankedPrize {
	defaultPrize := &RankedPrize{
		PrizeName: quickestOwnGoal,
		Rankings:  make([]Rank, 0),
	}

	if s == nil {
		return defaultPrize
	}
	events := make([]matchEventWithTeams, 0)

	for _, match := range s.Tournament.Matches {
		if !match.Completed {
			continue
		}

		events = append(events, (&matchEventsExtractor{match: match}).ownGoals()...)
	}

	sort.SliceStable(events, func(i, j int) bool {
		// sort by minute (asc) then by offset (asc)
		switch {
		case events[i].Minute == events[j].Minute:
			return events[i].Offset < events[j].Offset
		default:
			return events[i].Minute < events[j].Minute
		}
	})

	rankings := make([]Rank, 0)

	for _, ev := range events {
		rankings = append(rankings, Rank{
			ImageURL:        ev.For.ImageURL,
			ParticipantName: getSummaryFromTeamAndParticipant(ev.For, s.Participants.GetByTeamID(ev.For.ID)),
			Value:           fmt.Sprintf("🙈 %s (vs %s %s)", ev.String(), ev.Against.Name, ev.Timestamp.Format("02/01")),
		})
	}

	return &RankedPrize{
		PrizeName: quickestOwnGoal,
		Rankings:  rankings,
	}
}

type matchEventWithTeams struct {
	MatchEvent
	Timestamp time.Time
	For       *Team
	Against   *Team
}

type matchEventsExtractor struct {
	match *Match
}

func (m *matchEventsExtractor) ownGoals() []matchEventWithTeams {
	events := make([]matchEventWithTeams, 0)
	timestamp := m.match.Timestamp
	home := m.match.Home
	away := m.match.Away

	for _, og := range home.OwnGoals {
		events = append(events, matchEventWithTeams{
			MatchEvent: og,
			Timestamp:  timestamp,
			For:        home.Team,
			Against:    away.Team,
		})
	}

	for _, og := range away.OwnGoals {
		events = append(events, matchEventWithTeams{
			MatchEvent: og,
			Timestamp:  timestamp,
			For:        away.Team,
			Against:    home.Team,
		})
	}

	return events
}

// TODO: prize - quickest red card

type RankedPrize struct {
	PrizeName string
	Rankings  []Rank
}

type Rank struct {
	Position        uint8  // numerical position of rank
	ImageURL        string // image url
	ParticipantName string // participant name
	Value           string // match minute or qty (e.g. "45'+2" or "2 goals")
}
