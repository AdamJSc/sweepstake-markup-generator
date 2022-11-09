package domain

import (
	"fmt"
	"sort"
)

const (
	// finalMatchID defines the id of the match considered to be the final
	finalMatchID       = "F"
	mostGoalsConceded  = "Most Goals Conceded"
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

// TODO: prize - most yellow cards
// TODO: prize - quickest own goal
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
