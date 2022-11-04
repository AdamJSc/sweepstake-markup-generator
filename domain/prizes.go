package domain

import "fmt"

const (
	// finalMatchID defines the id of the match considered to be the final
	finalMatchID       = "F"
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
	winnerName := getSummary(winningTeam, participant)

	return &OutrightPrize{
		PrizeName:       tournamentWinner,
		ParticipantName: winnerName,
		ImageURL:        winningTeam.ImageURL,
	}
}

func getSummary(team *Team, participant *Participant) string {
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
	participantSummary := getSummary(runnerUpTeam, participant)

	return &OutrightPrize{
		PrizeName:       tournamentRunnerUp,
		ParticipantName: participantSummary,
		ImageURL:        runnerUpTeam.ImageURL,
	}
}

// TODO: prize - most goals conceded
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
