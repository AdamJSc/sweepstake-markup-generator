package domain_test

import (
	"testing"
	"time"

	"github.com/sweepstake-markup-generator/domain"
)

const (
	mostGoalsConceded  = "Most Goals Conceded"
	mostYellowCards    = "Most Yellow Cards"
	quickestOwnGoal    = "Quickest Own Goal"
	quickestRedCard    = "Quickest Red Card"
	tournamentRunnerUp = "Tournament Runner-Up"
	tournamentWinner   = "Tournament Winner"
)

var (
	date1        = time.Date(2018, 5, 26, 14, 0, 0, 0, tz)
	date2        = date1.Add(24 * time.Hour)
	date3        = date1.Add(48 * time.Hour)
	participantA = &domain.Participant{TeamID: "teamA", Name: "Marc Pugh"}
	participantB = &domain.Participant{TeamID: "teamB", Name: "Steve Fletcher"}
	participantC = &domain.Participant{TeamID: "teamC", Name: "Brett Pitman"}
	participantD = &domain.Participant{TeamID: "teamD", Name: "Shaun McDonald"}
	teamA        = &domain.Team{ID: "teamA", Name: "Team A", ImageURL: "http://teamA.jpg"}
	teamB        = &domain.Team{ID: "teamB", Name: "Team B", ImageURL: "http://teamB.jpg"}
	teamC        = &domain.Team{ID: "teamC", Name: "Team C", ImageURL: "http://teamC.jpg"}
	teamD        = &domain.Team{ID: "teamD", Name: "Team D", ImageURL: "http://teamD.jpg"}
	tz           = time.FixedZone("Europe/London", 3600)
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
						Value:           "\U0001F7E8️ 6",
					},
					{
						Position:        2,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "\U0001F7E8️ 2",
					},
					{
						Position:        3,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "\U0001F7E8️ 1",
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

func TestQuickestOwnGoal(t *testing.T) {
	defaultPrize := &domain.RankedPrize{PrizeName: quickestOwnGoal, Rankings: []domain.Rank{}}

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
						{
							Completed: true,
							Timestamp: date1,
							Home: domain.MatchCompetitor{
								Team: teamA,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "Lennon",
										Minute: 90,
										Offset: 1,
									},
									{
										Name:   "McCartney",
										Minute: 2,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "G.Harrison",
										Minute: 90,
									},
								},
							},
						},
						// not completed, should be ignored
						{
							// completed is false
							Timestamp: date2,
							Home: domain.MatchCompetitor{
								Team: teamA,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "Starr",
										Minute: 123,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "B.Epstein",
										Minute: 123,
									},
								},
							},
						}, {
							Completed: true,
							Timestamp: date3,
							Home: domain.MatchCompetitor{
								Team: teamC,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "Johnny",
										Minute: 46,
									},
									{
										Name:   "Joey",
										Minute: 45,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamD,
								OwnGoals: []domain.MatchEvent{
									{
										Name:   "DeeDee",
										Minute: 45,
										Offset: 4,
									},
									{
										Name:   "Tommy",
										Minute: 45,
										Offset: 5,
									},
								},
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: &domain.RankedPrize{
				PrizeName: quickestOwnGoal,
				Rankings: []domain.Rank{
					{
						Position:        1,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "🙈 2' McCartney (vs Team B 26/05)",
					},
					{
						Position:        2,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "🙈 45' Joey (vs Team D 28/05)",
					},
					{
						Position:        3,
						ImageURL:        "http://teamD.jpg",
						ParticipantName: "Shaun McDonald (Team D)",
						Value:           "🙈 45'+4 DeeDee (vs Team C 28/05)",
					},
					{
						Position:        4,
						ImageURL:        "http://teamD.jpg",
						ParticipantName: "Shaun McDonald (Team D)",
						Value:           "🙈 45'+5 Tommy (vs Team C 28/05)",
					},
					{
						Position:        5,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "🙈 46' Johnny (vs Team D 28/05)",
					},
					{
						Position:        6,
						ImageURL:        "http://teamB.jpg",
						ParticipantName: "Steve Fletcher (Team B)",
						Value:           "🙈 90' G.Harrison (vs Team A 26/05)",
					},
					{
						Position:        7,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "🙈 90'+1 Lennon (vs Team B 26/05)",
					},
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
			gotPrize := domain.QuickestOwnGoal(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}

func TestQuickestRedCard(t *testing.T) {
	defaultPrize := &domain.RankedPrize{PrizeName: quickestRedCard, Rankings: []domain.Rank{}}

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
						{
							Completed: true,
							Timestamp: date1,
							Home: domain.MatchCompetitor{
								Team: teamA,
								RedCards: []domain.MatchEvent{
									{
										Name:   "Lennon",
										Minute: 90,
										Offset: 1,
									},
									{
										Name:   "McCartney",
										Minute: 2,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
								RedCards: []domain.MatchEvent{
									{
										Name:   "G.Harrison",
										Minute: 90,
									},
								},
							},
						},
						// not completed, should be ignored
						{
							// completed is false
							Timestamp: date2,
							Home: domain.MatchCompetitor{
								Team: teamA,
								RedCards: []domain.MatchEvent{
									{
										Name:   "Starr",
										Minute: 123,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamB,
								RedCards: []domain.MatchEvent{
									{
										Name:   "B.Epstein",
										Minute: 123,
									},
								},
							},
						}, {
							Completed: true,
							Timestamp: date3,
							Home: domain.MatchCompetitor{
								Team: teamC,
								RedCards: []domain.MatchEvent{
									{
										Name:   "Johnny",
										Minute: 46,
									},
									{
										Name:   "Joey",
										Minute: 45,
									},
								},
							},
							Away: domain.MatchCompetitor{
								Team: teamD,
								RedCards: []domain.MatchEvent{
									{
										Name:   "DeeDee",
										Minute: 45,
										Offset: 4,
									},
									{
										Name:   "Tommy",
										Minute: 45,
										Offset: 5,
									},
								},
							},
						},
					},
				},
				Participants: participants,
			},
			wantPrize: &domain.RankedPrize{
				PrizeName: quickestRedCard,
				Rankings: []domain.Rank{
					{
						Position:        1,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "🟥 2' McCartney (vs Team B 26/05)",
					},
					{
						Position:        2,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "🟥 45' Joey (vs Team D 28/05)",
					},
					{
						Position:        3,
						ImageURL:        "http://teamD.jpg",
						ParticipantName: "Shaun McDonald (Team D)",
						Value:           "🟥 45'+4 DeeDee (vs Team C 28/05)",
					},
					{
						Position:        4,
						ImageURL:        "http://teamD.jpg",
						ParticipantName: "Shaun McDonald (Team D)",
						Value:           "🟥 45'+5 Tommy (vs Team C 28/05)",
					},
					{
						Position:        5,
						ImageURL:        "http://teamC.jpg",
						ParticipantName: "Brett Pitman (Team C)",
						Value:           "🟥 46' Johnny (vs Team D 28/05)",
					},
					{
						Position:        6,
						ImageURL:        "http://teamB.jpg",
						ParticipantName: "Steve Fletcher (Team B)",
						Value:           "🟥 90' G.Harrison (vs Team A 26/05)",
					},
					{
						Position:        7,
						ImageURL:        "http://teamA.jpg",
						ParticipantName: "Marc Pugh (Team A)",
						Value:           "🟥 90'+1 Lennon (vs Team B 26/05)",
					},
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
			gotPrize := domain.QuickestRedCard(tc.sweepstake)
			cmpDiff(t, tc.wantPrize, gotPrize)
		})
	}
}
