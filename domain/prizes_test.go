package domain_test

import (
	"testing"

	"github.com/sweepstake.adamjs.net/domain"
)

const (
	mostGoalsConceded  = "Most Goals Conceded"
	mostYellowCards    = "Most Yellow Cards"
	tournamentRunnerUp = "Tournament Runner-Up"
	tournamentWinner   = "Tournament Winner"
)

var (
	participantA = &domain.Participant{TeamID: "teamA", Name: "Marc Pugh"}
	participantB = &domain.Participant{TeamID: "teamB", Name: "Steve Fletcher"}
	participantC = &domain.Participant{TeamID: "teamC", Name: "Brett Pitman"}
	participantD = &domain.Participant{TeamID: "teamD", Name: "Shaun McDonald"}
	teamA        = &domain.Team{ID: "teamA", Name: "Team A", ImageURL: "http://teamA.jpg"}
	teamB        = &domain.Team{ID: "teamB", Name: "Team B", ImageURL: "http://teamB.jpg"}
	teamC        = &domain.Team{ID: "teamC", Name: "Team C", ImageURL: "http://teamC.jpg"}
	teamD        = &domain.Team{ID: "teamD", Name: "Team D", ImageURL: "http://teamD.jpg"}
)

func TestTournamentWinner(t *testing.T) {
	defaultPrize := &domain.OutrightPrize{PrizeName: tournamentWinner, ParticipantName: "TBC"}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  *domain.OutrightPrize
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
				Participants: domain.ParticipantCollection{participantA},
			},
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentWinner,
				ParticipantName: "Marc Pugh (Team A)",
				ImageURL:        "http://teamA.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentWinner,
				ParticipantName: "Team A",
				ImageURL:        "http://teamA.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentWinner,
				ParticipantName: "Team A",
				ImageURL:        "http://teamA.jpg",
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
				Participants: domain.ParticipantCollection{participantA},
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
				Participants: domain.ParticipantCollection{participantA},
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
				Participants: domain.ParticipantCollection{participantA},
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
	defaultPrize := &domain.OutrightPrize{PrizeName: tournamentRunnerUp, ParticipantName: "TBC"}
	participants := domain.ParticipantCollection{participantA, participantB}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  *domain.OutrightPrize
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Steve Fletcher (Team B)",
				ImageURL:        "http://teamB.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Marc Pugh (Team A)",
				ImageURL:        "http://teamA.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Team B",
				ImageURL:        "http://teamB.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Team A",
				ImageURL:        "http://teamA.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Team B",
				ImageURL:        "http://teamB.jpg",
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
			wantPrize: &domain.OutrightPrize{
				PrizeName:       tournamentRunnerUp,
				ParticipantName: "Team A",
				ImageURL:        "http://teamA.jpg",
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

func TestMostGoalsConceded(t *testing.T) {
	defaultPrize := &domain.RankedPrize{PrizeName: mostGoalsConceded, Rankings: []domain.Rank{}}

	teams := domain.TeamCollection{teamA, teamB, teamC, teamD}
	participants := domain.ParticipantCollection{participantA, participantB, participantC, participantD}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  *domain.RankedPrize
	}{
		{
			name: "valid sweepstake must produce the expected rankings",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Teams: teams,
					Matches: domain.MatchCollection{
						// teamA = 1 (1)
						// teamB = 2 (2)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:  teamA,
								Goals: 2,
							},
							Away: domain.MatchCompetitor{
								Team:  teamB,
								Goals: 1,
							},
						},
						// not completed, should be ignored
						{
							// completed is false
							Home: domain.MatchCompetitor{
								Team:  teamA,
								Goals: 99,
							},
							Away: domain.MatchCompetitor{
								Team:  teamB,
								Goals: 99,
							},
						},
						// teamB = 3 (5)
						// teamC = 2 (2)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:  teamB,
								Goals: 2,
							},
							Away: domain.MatchCompetitor{
								Team:  teamC,
								Goals: 3,
							},
						},
						// teamB = 1 (6)
						// teamD = 0 (0)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:  teamB,
								Goals: 0,
							},
							Away: domain.MatchCompetitor{
								Team:  teamD,
								Goals: 1,
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: &domain.RankedPrize{
				PrizeName: mostGoalsConceded,
				Rankings: []domain.Rank{
					{
						Position:        1,
						ImageURL:        "http://teamB.jpg",
						ParticipantName: "Steve Fletcher (Team B)",
						Value:           "⚽️ 6",
					},
					{
						Position:        2,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "⚽️ 2",
					},
					{
						Position:        3,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "⚽️ 1",
					},
					// teamD do not rank
				},
			},
		},
		{
			name:      "no sweepstake must return default prize",
			wantPrize: defaultPrize,
			// nil sweepstake
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotPrize := domain.MostGoalsConceded(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}

func TestMostYellowCards(t *testing.T) {
	defaultPrize := &domain.RankedPrize{PrizeName: mostYellowCards, Rankings: []domain.Rank{}}

	teams := domain.TeamCollection{teamA, teamB, teamC, teamD}
	participants := domain.ParticipantCollection{participantA, participantB, participantC, participantD}

	tt := []struct {
		name       string
		sweepstake *domain.Sweepstake
		wantPrize  *domain.RankedPrize
	}{
		{
			name: "valid sweepstake must produce the expected rankings",
			sweepstake: &domain.Sweepstake{
				Tournament: &domain.Tournament{
					Teams: teams,
					Matches: domain.MatchCollection{
						// teamA = 1 (1)
						// teamB = 2 (2)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:        teamA,
								YellowCards: 1,
							},
							Away: domain.MatchCompetitor{
								Team:        teamB,
								YellowCards: 2,
							},
						},
						// not completed, should be ignored
						{
							// completed is false
							Home: domain.MatchCompetitor{
								Team:        teamA,
								YellowCards: 99,
							},
							Away: domain.MatchCompetitor{
								Team:        teamB,
								YellowCards: 99,
							},
						},
						// teamB = 3 (5)
						// teamC = 2 (2)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:        teamB,
								YellowCards: 3,
							},
							Away: domain.MatchCompetitor{
								Team:        teamC,
								YellowCards: 2,
							},
						},
						// teamB = 1 (6)
						// teamD = 0 (0)
						{
							Completed: true,
							Home: domain.MatchCompetitor{
								Team:        teamB,
								YellowCards: 1,
							},
							Away: domain.MatchCompetitor{
								Team:        teamD,
								YellowCards: 0,
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: &domain.RankedPrize{
				PrizeName: mostYellowCards,
				Rankings: []domain.Rank{
					{
						Position:        1,
						ImageURL:        "http://teamB.jpg",
						ParticipantName: "Steve Fletcher (Team B)",
						Value:           "⚽️ 6",
					},
					{
						Position:        2,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "⚽️ 2",
					},
					{
						Position:        3,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "⚽️ 1",
					},
					// teamD do not rank
				},
			},
		},
		{
			name:      "no sweepstake must return default prize",
			wantPrize: defaultPrize,
			// nil sweepstake
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gotPrize := domain.MostYellowCards(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}
