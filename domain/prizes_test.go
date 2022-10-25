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

func TestTournamentRunnerUp(t *testing.T) {
	defaultPrize := domain.OutrightPrize{WinnerName: "TBC"}

	teamA := &domain.Team{ID: "teamA", Name: "Team A", ImageURL: "http://teamA.jpg"}
	teamB := &domain.Team{ID: "teamB", Name: "Team B", ImageURL: "http://teamB.jpg"}

	participants := domain.ParticipantCollection{
		{
			TeamID: "teamA",
			Name:   "Marc Pugh",
		},
		{
			TeamID: "teamB",
			Name:   "Steve Fletcher",
		},
	}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  domain.OutrightPrize
	}{
		{
			name: "completed final match with confirmed winning teamA and participant name must return prize with participant name and team name",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Steve Fletcher (Team B)",
				ImageURL:   "http://teamB.jpg",
			},
		},
		{
			name: "completed final match with confirmed winning teamB and participant name must return prize with participant name and team name",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamB,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Marc Pugh (Team A)",
				ImageURL:   "http://teamA.jpg",
			},
		},
		{
			name: "completed final match with confirmed winning teamA and no participant name must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						// no name
					},
					{
						TeamID: "teamB",
						// no name
					},
				},
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Team B",
				ImageURL:   "http://teamB.jpg",
			},
		},
		{
			name: "completed final match with confirmed winning teamB and no participant name must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamB,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				Participants: domain.ParticipantCollection{
					{
						TeamID: "teamA",
						// no name
					},
					{
						TeamID: "teamB",
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
			name: "completed final match with confirmed winning teamA and no participant must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamA,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				// no participants
			},
			wantPrize: domain.OutrightPrize{
				WinnerName: "Team B",
				ImageURL:   "http://teamB.jpg",
			},
		},
		{
			name: "completed final match with confirmed winning teamB and no participant must return prize with team name only",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Matches: domain.MatchCollection{
						{
							ID:        "F",
							Completed: true,
							Winner:    teamB,
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
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
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
							// completed is false
						},
					},
				},
				Participants: participants,
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
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
							// no winner
						},
					},
				},
				Participants: participants,
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
							Home: domain.MatchCompetitor{
								Team: teamA,
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
							},
						},
					},
				},
				Participants: participants,
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
			gotPrize := domain.TournamentRunnerUp(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}
