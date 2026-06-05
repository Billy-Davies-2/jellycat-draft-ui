package dal

import (
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

func TestCalculateCurrentPickAddsDraftIntel(t *testing.T) {
	state := &models.DraftState{
		Players: []models.Player{
			{ID: "drafted-star", Name: "Drafted Star", Position: "CC", Team: "Test", Points: 400, CuddlePoints: 70, Tier: models.TierS, Drafted: true},
			{ID: "best-fit", Name: "Best Fit", Position: "SS", Team: "Test", Points: 310, CuddlePoints: 65, Tier: models.TierS},
			{ID: "backup-fit", Name: "Backup Fit", Position: "CC", Team: "Test", Points: 220, CuddlePoints: 50, Tier: models.TierB},
		},
		Teams: []models.Team{
			{
				ID:      "team-1",
				Name:    "Test Team",
				Owner:   "Owner",
				Mascot:  "TT",
				Players: []models.Player{{ID: "rostered", Name: "Rostered", Position: "CC", Team: "Test", Points: 120, CuddlePoints: 50, Tier: models.TierB}},
			},
		},
		Settings: models.DefaultDraftSettings(),
	}

	CalculateCurrentPick(state, state.Players)

	if state.SuggestedPick == nil {
		t.Fatal("expected suggested pick")
	}
	if state.SuggestedPick.Player.Drafted {
		t.Fatalf("suggested pick should not be drafted: %+v", state.SuggestedPick.Player)
	}
	if !state.SuggestedPick.Player.Analytics.Suggested {
		t.Fatal("suggested player should be marked on its analytics")
	}
	if state.SuggestedPick.Confidence <= 0 || state.SuggestedPick.Confidence > 99 {
		t.Fatalf("suggested pick confidence out of range: %d", state.SuggestedPick.Confidence)
	}
	if state.SuggestedPick.Reason == "" {
		t.Fatal("suggested pick should include a reason")
	}

	for _, player := range state.Players {
		if player.Analytics.PickScore <= 0 || player.Analytics.PickScore > 99 {
			t.Fatalf("player %s pick score out of range: %d", player.ID, player.Analytics.PickScore)
		}
		if len(player.Analytics.Sparkline) != 6 {
			t.Fatalf("player %s sparkline length = %d, want 6", player.ID, len(player.Analytics.Sparkline))
		}
		if player.Analytics.TrendLabel == "" || player.Analytics.Label == "" || player.Analytics.Reason == "" {
			t.Fatalf("player %s missing display analytics: %+v", player.ID, player.Analytics)
		}
	}
}

func TestDraftIntelUsesCuddlePointsWhenPointsAreZero(t *testing.T) {
	lowGrade := models.Player{ID: "low-grade", Name: "Low Grade", Position: "CC", Team: "Test", Points: 0, CuddlePoints: 68, Tier: models.TierC}
	highGrade := models.Player{ID: "high-grade", Name: "High Grade", Position: "SS", Team: "Test", Points: 0, CuddlePoints: 98, Tier: models.TierS}

	lowAnalytics := buildPlayerAnalytics(lowGrade, models.DraftModeStandard, 1, 0, nil, false)
	highAnalytics := buildPlayerAnalytics(highGrade, models.DraftModeStandard, 1, 0, nil, false)

	if lowAnalytics.ValueScore == 35 && highAnalytics.ValueScore == 35 {
		t.Fatalf("value scores should not both clamp to 35 when cuddle points are set: low=%d high=%d", lowAnalytics.ValueScore, highAnalytics.ValueScore)
	}
	if highAnalytics.ValueScore <= lowAnalytics.ValueScore {
		t.Fatalf("higher cuddle grade should have better value score: low=%d high=%d", lowAnalytics.ValueScore, highAnalytics.ValueScore)
	}
	if highAnalytics.NeedFit == lowAnalytics.NeedFit {
		t.Fatalf("fallback fit scores should vary by player seed: low=%d high=%d", lowAnalytics.NeedFit, highAnalytics.NeedFit)
	}
}

func TestBingoBoardScalesToPlayerCount(t *testing.T) {
	state := &models.DraftState{
		Players:  testPlayers(15, 14),
		Teams:    testTeams(6),
		Settings: models.DraftSettingsForMode(models.DraftModeBingo),
	}

	CalculateCurrentPick(state, state.Players)

	if len(state.BingoBoard) != 15 {
		t.Fatalf("bingo board length = %d, want 15", len(state.BingoBoard))
	}
	if state.CurrentBingoPrompt == "" {
		t.Fatal("expected current bingo prompt")
	}
	if !state.BingoBoard[14].Active {
		t.Fatalf("expected final square to be active: %+v", state.BingoBoard[14])
	}
	for index := 0; index < 14; index++ {
		if !state.BingoBoard[index].Completed {
			t.Fatalf("expected square %d to be completed", index+1)
		}
	}
}

func TestWheelSlotsScaleToPartialFinalRound(t *testing.T) {
	state := &models.DraftState{
		Players:  testPlayers(15, 12),
		Teams:    testTeams(6),
		Settings: models.DraftSettingsForMode(models.DraftModeWheel),
	}

	CalculateCurrentPick(state, state.Players)

	if state.CurrentRound != 3 {
		t.Fatalf("current round = %d, want 3", state.CurrentRound)
	}
	if len(state.WheelSlots) != 3 {
		t.Fatalf("wheel slot length = %d, want 3", len(state.WheelSlots))
	}
	if len(state.DraftOrder) != 15 {
		t.Fatalf("draft order length = %d, want 15", len(state.DraftOrder))
	}
	if state.DraftOrder[len(state.DraftOrder)-1].Pick != 15 {
		t.Fatalf("last pick = %d, want 15", state.DraftOrder[len(state.DraftOrder)-1].Pick)
	}
	if !state.WheelSlots[0].Active {
		t.Fatalf("expected first partial-round slot to be active: %+v", state.WheelSlots)
	}
}

func testPlayers(total, drafted int) []models.Player {
	players := make([]models.Player, 0, total)
	positions := []string{"CC", "SS", "HH", "CH"}
	for index := 0; index < total; index++ {
		players = append(players, models.Player{
			ID:           string(rune('a' + index)),
			Name:         "Jellycat",
			Position:     positions[index%len(positions)],
			Team:         "Test",
			Points:       200 + index,
			CuddlePoints: 50 + (index % 20),
			Tier:         models.TierA,
			Drafted:      index < drafted,
		})
	}
	return players
}

func testTeams(total int) []models.Team {
	teams := make([]models.Team, 0, total)
	for index := 0; index < total; index++ {
		teams = append(teams, models.Team{
			ID:     string(rune('A' + index)),
			Name:   "Team",
			Mascot: "JT",
		})
	}
	return teams
}
