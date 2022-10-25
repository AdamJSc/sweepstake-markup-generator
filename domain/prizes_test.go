package domain_test

import (
	"testing"

	"github.com/sweepstake.adamjs.net/domain"
)

func TestTournamentWinner(t *testing.T) {
	defaultPrize := domain.OutrightPrize{WinnerName: "TBC"}
	teamA := &domain.Team{ID: "teamA", Name: "Team A", ImageURL: "http://teamA.jpg"}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  domain.OutrightPrize
	}{
		{
			name: "completed final match with winning team and participant name must return prize with participant name and team name",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						Name:   "Marc Pugh",
					},
				},
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Marc Pugh (Team A)",
				ImageURL:   "http://teamA.jpg",
			},
		},
		{
			name: "completed final match with winning team and no participant name must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						// no name
					},
				},
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Team A",
				ImageURL:   "http://teamA.jpg",
			},
		},
		{
			name: "completed final match with winning team and no participant must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
						},
					},
				},
				// no participants
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Team A",
				ImageURL:   "http://teamA.jpg",
			},
		},
		{
			name: "final match that has not yet completed must return default prize",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:     "F",
							Winner: teamA,
							// completed is false
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						Name:   "Marc Pugh",
					},
				},
			},
			wantPrize: defaultPrize,
		},
		{
			name: "final match that has no winner must return default prize",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							// no winner
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						Name:   "Marc Pugh",
					},
				},
			},
			wantPrize: defaultPrize,
		},
		{
			name: "no final must return default prize",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "NOT-F",
							Completed: true,
							Winner:    teamA,
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						Name:   "Marc Pugh",
					},
				},
			},
			wantPrize: defaultPrize,
		},
		{
			name:      "no sweepstake must return default prize",
			wantPrize: defaultPrize,
			// nil sweepstake
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotPrize := domain.TournamentWinner(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}
