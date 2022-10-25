package domain

import "fmt"

// finalMatchID defines the id of the match considered to be the final
const finalMatchID = "F"

// defaultOutright defines a default outright prize
var defaultOutright = OutrightPrize{
	WinnerName: "TBC",
}

// OutrightPrize represents a prize with a single outright winner
type OutrightPrize struct {
	WinnerName string
	ImageURL   string
}

// OutrightPrizeGenerator defines a function that generates an outright prize from the provided Sweepstake
type OutrightPrizeGenerator func(sweepstake *Sweepstake) OutrightPrize

// TournamentWinner determines the winner of the provided Sweepstake
var TournamentWinner = func(s *Sweepstake) OutrightPrize {
	if s == nil {
		return defaultOutright
	}

	// get match winner
	winningTeam := s.Tournament.Matches.GetWinnerByMatchID(finalMatchID)
	if winningTeam == nil {
		return defaultOutright
	}

	// get participant who represents the match winner
	participant := s.Participants.GetByTeamID(winningTeam.ID)
	winnerName := getSummary(winningTeam, participant)

	return OutrightPrize{
		WinnerName: winnerName,
		ImageURL:   winningTeam.ImageURL,
	}
}

func getSummary(team *Team, participant *Participant) string {
	if participant == nil || participant.Name == "" {
		return team.Name
	}

	return fmt.Sprintf("%s (%s)", participant.Name, team.Name)
}
