package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type Sweepstake struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ImageURL     string `json:"image_url"`
	Tournament   *Tournament
	Participants ParticipantCollection `json:"participants"`
	Prizes       PrizeSettings         `json:"prizes"`
	Build        bool                  `json:"build"`
}

func (s *Sweepstake) GenerateMarkup() ([]byte, error) {
	// TODO: test this method using actual tournament data to check for regressions
	buf := &bytes.Buffer{}

	// generate outright prize data
	var winner, runnerUp *OutrightPrize
	if s.Prizes.Winner {
		winner = TournamentWinner(s)
	}
	if s.Prizes.RunnerUp {
		runnerUp = TournamentRunnerUp(s)
	}

	// generate ranked prize data
	var mostGoalsConceded, mostYellowCards, quickestOwnGoal, quickestRedCard *RankedPrize
	if s.Prizes.MostGoalsConceded {
		// TODO: replace with most goals conceded prize generator
		mostGoalsConceded = &RankedPrize{PrizeName: "Most Goals Conceded", Rankings: []Rank{
			{
				Position:        1,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/1/1a/Flag_of_Argentina.svg",
				ParticipantName: "Mr Argentina (ARG)",
				Value:           "⚽️ 10",
			},
			{
				Position:        2,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_Australia_%28converted%29.svg",
				ParticipantName: "Mr Australia (AUS)",
				Value:           "⚽️ 5",
			},
			{
				Position:        3,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Croatia.svg",
				ParticipantName: "Mr Croatia (HRV)",
				Value:           "⚽️ 2",
			},
		}}
	}
	if s.Prizes.MostYellowCards {
		// TODO: replace with most yellow cards prize generator
		mostYellowCards = &RankedPrize{PrizeName: "Most Yellow Cards"}
	}
	if s.Prizes.QuickestOwnGoal {
		// TODO: replace with quickest own goal prize generator
		quickestOwnGoal = &RankedPrize{PrizeName: "Quickest Own Goal", Rankings: []Rank{
			{
				Position:        1,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/1/1a/Flag_of_Argentina.svg",
				ParticipantName: "Mr Argentina (ARG)",
				Value:           "⚽️ 45'+2 (vs Canada)",
			},
			{
				Position:        2,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/8/88/Flag_of_Australia_%28converted%29.svg",
				ParticipantName: "Mr Australia (AUS)",
				Value:           "⚽️ 76' (vs Canada)",
			},
			{
				Position:        3,
				ImageURL:        "https://upload.wikimedia.org/wikipedia/commons/1/1b/Flag_of_Croatia.svg",
				ParticipantName: "Mr Croatia (HRV)",
				Value:           "⚽️ 87' (vs Morocco)",
			},
		}}
	}
	if s.Prizes.QuickestRedCard {
		// TODO: replace with quickest red card prize generator
		quickestRedCard = &RankedPrize{PrizeName: "Quickest Red Card"}
	}

	// set title as sweepstake name, fallback to tournament name if missing
	title := s.Name
	if title == "" {
		title = s.Tournament.Name
	}

	// set image url as sweepstake, fallback to tournament if missing
	imageURL := s.ImageURL
	if imageURL == "" {
		imageURL = s.Tournament.ImageURL
	}

	type prizeData struct {
		Winner            *OutrightPrize
		RunnerUp          *OutrightPrize
		MostGoalsConceded *RankedPrize
		MostYellowCards   *RankedPrize
		QuickestOwnGoal   *RankedPrize
		QuickestRedCard   *RankedPrize
	}

	data := struct {
		Title      string
		ImageURL   string
		Prizes     prizeData
		Sweepstake *Sweepstake
	}{
		Title:    title,
		ImageURL: imageURL,
		Prizes: prizeData{
			Winner:            winner,
			RunnerUp:          runnerUp,
			MostGoalsConceded: mostGoalsConceded,
			MostYellowCards:   mostYellowCards,
			QuickestOwnGoal:   quickestOwnGoal,
			QuickestRedCard:   quickestRedCard,
		},
		Sweepstake: s,
	}

	if err := s.Tournament.Template.ExecuteTemplate(buf, "tpl", data); err != nil {
		return nil, fmt.Errorf("cannot execute template: %w", err)
	}

	return buf.Bytes(), nil
}

type Participant struct {
	TeamID string `json:"team_id"`
	Name   string `json:"participant_name"`
}

type ParticipantCollection []*Participant

func (pc ParticipantCollection) GetByTeamID(id string) *Participant {
	for _, participant := range pc {
		if participant != nil && participant.TeamID == id {
			return participant
		}
	}

	return nil
}

type PrizeSettings struct {
	Winner            bool `json:"winner"`
	RunnerUp          bool `json:"runner_up"`
	MostGoalsConceded bool `json:"most_goals_conceded"`
	MostYellowCards   bool `json:"most_yellow_cards"`
	QuickestOwnGoal   bool `json:"quickest_own_goal"`
	QuickestRedCard   bool `json:"quickest_red_card"`
}

type SweepstakeCollection []*Sweepstake

type SweepstakesJSONLoader struct {
	fSys        fs.FS
	tournaments TournamentCollection
	configPath  string
}

func (s *SweepstakesJSONLoader) WithFileSystem(fSys fs.FS) *SweepstakesJSONLoader {
	s.fSys = fSys
	return s
}

func (s *SweepstakesJSONLoader) WithTournamentCollection(tournaments TournamentCollection) *SweepstakesJSONLoader {
	s.tournaments = tournaments
	return s
}

func (s *SweepstakesJSONLoader) WithConfigPath(path string) *SweepstakesJSONLoader {
	s.configPath = path
	return s
}

func (s *SweepstakesJSONLoader) init() error {
	if s.fSys == nil {
		s.fSys = defaultFileSystem
	}

	if s.tournaments == nil {
		return fmt.Errorf("tournaments: %w", ErrIsEmpty)
	}

	if s.configPath == "" {
		return fmt.Errorf("config path: %w", ErrIsEmpty)
	}

	return nil
}

func (s *SweepstakesJSONLoader) LoadSweepstakes(_ context.Context) (SweepstakeCollection, error) {
	if err := s.init(); err != nil {
		return nil, err
	}

	// read sweepstake config file
	rawConfigJSON, err := readFile(s.fSys, s.configPath)
	if err != nil {
		return nil, err
	}

	// parse as sweepstakes
	var content = &struct {
		Sweepstakes []struct {
			*Sweepstake
			TournamentID string `json:"tournament_id"`
		} `json:"sweepstakes"`
	}{}
	if err = json.Unmarshal(rawConfigJSON, content); err != nil {
		return nil, fmt.Errorf("cannot unmarshal sweepstakes: %w", err)
	}

	collection := make(SweepstakeCollection, 0)
	for idx := range content.Sweepstakes {
		sweepstake := content.Sweepstakes[idx].Sweepstake
		tournamentID := content.Sweepstakes[idx].TournamentID

		// inflate tournament
		tournament := s.tournaments.GetByID(tournamentID)
		if tournament == nil {
			return nil, fmt.Errorf("sweepstake index %d: tournament id '%s': %w", idx, tournamentID, ErrNotFound)
		}
		sweepstake.Tournament = tournament

		collection = append(collection, sweepstake)
	}

	return validateSweepstakes(collection)
}

func validateSweepstakes(sweepstakes SweepstakeCollection) (SweepstakeCollection, error) {
	ids := &sync.Map{}
	mErr := NewMultiError()

	for _, sweepstake := range sweepstakes {
		mErrIdx := mErr.WithPrefix(fmt.Sprintf("id '%s'", sweepstake.ID))

		// check if this sweepstake id already exists in the collection
		if _, ok := ids.Load(sweepstake.ID); ok {
			mErrIdx.Add(ErrIsDuplicate)
		}
		ids.Store(sweepstake.ID, struct{}{})

		// run remaining validation
		validateSweepstake(sweepstake, mErr)
	}

	if !mErr.IsEmpty() {
		return nil, mErr
	}

	return sweepstakes, nil
}

func validateSweepstake(sweepstake *Sweepstake, mErr MultiError) *Sweepstake {
	sweepstake.ID = strings.Trim(sweepstake.ID, " ")
	sweepstake.Name = strings.Trim(sweepstake.Name, " ")
	sweepstake.ImageURL = strings.Trim(sweepstake.ImageURL, " ")

	if sweepstake.ID == "" {
		mErr.Add(fmt.Errorf("id: %w", ErrIsEmpty))
	}

	if sweepstake.Name == "" {
		mErr.Add(fmt.Errorf("name: %w", ErrIsEmpty))
	}

	if sweepstake.ImageURL == "" {
		mErr.Add(fmt.Errorf("image url: %w", ErrIsEmpty))
	}

	audit := &teamsAudit{teams: sweepstake.Tournament.Teams}
	for idx, participant := range sweepstake.Participants {
		participant.TeamID = strings.Trim(participant.TeamID, " ")
		participant.Name = strings.Trim(participant.Name, " ")

		mErrIdx := mErr.WithPrefix(fmt.Sprintf("participant index %d", idx))

		if ok := audit.ack(&Team{ID: participant.TeamID}); !ok {
			mErrIdx.Add(fmt.Errorf("unrecognised participant team id: %s", participant.TeamID))
		}
	}

	audit.validate(mErr, true)

	return sweepstake
}
