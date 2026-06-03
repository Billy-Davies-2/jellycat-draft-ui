package dal

import (
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

var bingoPrompts = []string{
	"Tiny but mighty",
	"Garden party pick",
	"Ocean friend",
	"Snack squad",
	"Cloud-soft favorite",
	"Wild card mascot",
	"Pastel power",
	"Bedtime captain",
	"Best name wins",
	"Round one rival",
	"Cozy underdog",
	"Collectors' choice",
	"Center square star",
	"Birthday-table energy",
	"Travel buddy",
	"Highest cuddle points",
	"Lowest sleeper pick",
	"Fantasy creature",
	"Farmyard friend",
	"Tiny desk legend",
	"Photo-ready pick",
	"Fan chant moment",
	"Theme team fit",
	"Wildcard trade bait",
	"Final flourish",
}

// CalculateCurrentPick enriches draft state with current turn and mode-specific UI data.
func CalculateCurrentPick(state *models.DraftState, players []models.Player) {
	state.Settings = models.DraftSettingsForMode(state.Settings.Mode)

	totalDrafted := countDraftedPlayers(players)
	totalPlayers := len(players)
	teamCount := len(state.Teams)
	defer func() {
		applyDraftIntel(state, totalDrafted)
	}()

	state.CurrentPick = totalDrafted + 1
	state.CurrentRound = 0
	state.PickInRound = 0
	state.CurrentTeamID = ""
	state.CurrentTeamName = ""
	state.DraftOrder = buildDraftOrder(state.Teams, state.Settings.Mode, totalDrafted, totalPlayers)
	state.BingoBoard = nil
	state.CurrentBingoPrompt = ""
	state.WheelSlots = nil
	state.SuggestedPick = nil

	if teamCount == 0 || totalPlayers == 0 || totalDrafted >= totalPlayers {
		state.CurrentPick = totalDrafted
		return
	}

	state.CurrentRound = (totalDrafted / teamCount) + 1
	state.PickInRound = (totalDrafted % teamCount) + 1

	team := expectedTeamForPick(state.Teams, state.Settings.Mode, totalDrafted)
	if team != nil {
		state.CurrentTeamID = team.ID
		state.CurrentTeamName = team.Name
	}

	if state.Settings.Mode == models.DraftModeBingo {
		state.BingoBoard = buildBingoBoard(totalDrafted, totalPlayers)
		state.CurrentBingoPrompt = currentBingoPrompt(totalDrafted, totalPlayers)
	}

	if state.Settings.Mode == models.DraftModeWheel {
		state.WheelSlots = buildWheelSlots(state.Teams, totalDrafted, totalPlayers)
	}
}

func applyDraftIntel(state *models.DraftState, totalDrafted int) {
	positionCounts, hasCurrentTeam := positionCountsForTeam(state.Teams, state.CurrentTeamID)
	analyticsByID := make(map[string]models.PlayerAnalytics, len(state.Players))
	bestPlayerIndex := -1
	bestScore := -1

	for playerIndex := range state.Players {
		analytics := buildPlayerAnalytics(state.Players[playerIndex], state.Settings.Mode, state.CurrentRound, totalDrafted, positionCounts, hasCurrentTeam)
		state.Players[playerIndex].Analytics = analytics
		analyticsByID[state.Players[playerIndex].ID] = analytics

		if state.CurrentTeamID != "" && !state.Players[playerIndex].Drafted && analytics.PickScore > bestScore {
			bestScore = analytics.PickScore
			bestPlayerIndex = playerIndex
		}
	}

	if bestPlayerIndex >= 0 {
		state.Players[bestPlayerIndex].Analytics.Suggested = true
		analyticsByID[state.Players[bestPlayerIndex].ID] = state.Players[bestPlayerIndex].Analytics
		state.SuggestedPick = &models.DraftRecommendation{
			Player:     state.Players[bestPlayerIndex],
			Reason:     state.Players[bestPlayerIndex].Analytics.Reason,
			Confidence: state.Players[bestPlayerIndex].Analytics.PickScore,
			ModelLabel: "Draft Intel",
		}
	}

	for teamIndex := range state.Teams {
		for playerIndex := range state.Teams[teamIndex].Players {
			player := &state.Teams[teamIndex].Players[playerIndex]
			if analytics, ok := analyticsByID[player.ID]; ok {
				player.Analytics = analytics
				continue
			}

			player.Analytics = buildPlayerAnalytics(*player, state.Settings.Mode, state.CurrentRound, totalDrafted, positionCounts, hasCurrentTeam)
		}
	}
}

func positionCountsForTeam(teams []models.Team, teamID string) (map[string]int, bool) {
	positionCounts := map[string]int{}
	if teamID == "" {
		return positionCounts, false
	}

	for _, team := range teams {
		if team.ID != teamID {
			continue
		}

		for _, player := range team.Players {
			positionCounts[player.Position]++
		}
		return positionCounts, true
	}

	return positionCounts, false
}

func buildPlayerAnalytics(player models.Player, mode models.DraftMode, currentRound, totalDrafted int, positionCounts map[string]int, hasCurrentTeam bool) models.PlayerAnalytics {
	if currentRound < 1 {
		currentRound = 1
	}

	seed := metricSeed(player.ID, player.Name, player.Position, player.Team, mode, totalDrafted)
	valueScore := clampInt((player.Points/4)+tierBoost(player.Tier)+(seed%13)-6, 35, 99)
	crowdHeat := clampInt((player.CuddlePoints*2/3)+38+((seed/7)%24)-12, 30, 99)
	needFit := needFitScore(player.Position, positionCounts, hasCurrentTeam)
	trendDelta := clampInt(((seed/13)%27)-8+tierTrend(player.Tier)+(currentRound-1), -9, 24)
	modeBoost := modeIntelBoost(mode, player, seed)
	pickScore := clampInt(((valueScore*4)+(crowdHeat*2)+(needFit*3)+((trendDelta+10)*2)+modeBoost)/11, 1, 99)

	analytics := models.PlayerAnalytics{
		PickScore:  pickScore,
		ValueScore: valueScore,
		CrowdHeat:  crowdHeat,
		NeedFit:    needFit,
		TrendDelta: trendDelta,
		TrendLabel: signedPercent(trendDelta),
		Sparkline:  buildSparkline(seed, valueScore, crowdHeat, trendDelta),
	}
	analytics.Label = analyticsLabel(analytics)
	analytics.Reason = analyticsReason(player, analytics)

	return analytics
}

func metricSeed(parts ...interface{}) int {
	hasher := fnv.New32a()
	for _, part := range parts {
		_, _ = fmt.Fprintf(hasher, "%v|", part)
	}
	return int(hasher.Sum32() % 100000)
}

func tierBoost(tier models.Tier) int {
	switch tier {
	case models.TierS:
		return 24
	case models.TierA:
		return 16
	case models.TierB:
		return 8
	default:
		return 2
	}
}

func tierTrend(tier models.Tier) int {
	switch tier {
	case models.TierS:
		return 4
	case models.TierA:
		return 2
	case models.TierB:
		return 0
	default:
		return -1
	}
}

func needFitScore(position string, positionCounts map[string]int, hasCurrentTeam bool) int {
	if !hasCurrentTeam {
		return 64
	}

	switch positionCounts[position] {
	case 0:
		return 94
	case 1:
		return 70
	default:
		return 44
	}
}

func modeIntelBoost(mode models.DraftMode, player models.Player, seed int) int {
	switch mode {
	case models.DraftModeBingo:
		if seed%3 == 0 {
			return 12
		}
		return 4
	case models.DraftModeWheel:
		return 6 + (len(player.Team) % 5)
	case models.DraftModeReverseSnake:
		return 5 + (seed % 6)
	default:
		return seed % 8
	}
}

func buildSparkline(seed, valueScore, crowdHeat, trendDelta int) []int {
	points := make([]int, 6)
	baseline := clampInt(((valueScore+crowdHeat)/2)-16, 18, 86)
	for index := range points {
		wiggle := ((seed >> uint(index*3)) % 17) - 8
		points[index] = clampInt(baseline+(index*trendDelta/3)+wiggle, 14, 96)
	}
	return points
}

func signedPercent(value int) string {
	if value > 0 {
		return fmt.Sprintf("+%d%%", value)
	}
	return fmt.Sprintf("%d%%", value)
}

func analyticsLabel(analytics models.PlayerAnalytics) string {
	switch {
	case analytics.PickScore >= 90:
		return "Best on board"
	case analytics.NeedFit >= 90:
		return "Roster fit"
	case analytics.CrowdHeat >= 86:
		return "Crowd favorite"
	case analytics.TrendDelta >= 14:
		return "Rising fast"
	case analytics.ValueScore >= 85:
		return "Value play"
	default:
		return "Steady pick"
	}
}

func analyticsReason(player models.Player, analytics models.PlayerAnalytics) string {
	switch {
	case analytics.NeedFit >= 90:
		return fmt.Sprintf("Fills the %s slot while keeping a %d pick score.", player.Position, analytics.PickScore)
	case analytics.CrowdHeat >= 86:
		return fmt.Sprintf("High crowd heat and a %s trend make this a lively board pick.", analytics.TrendLabel)
	case analytics.TrendDelta >= 14:
		return fmt.Sprintf("Trending %s with strong value for this round.", analytics.TrendLabel)
	case analytics.ValueScore >= 85:
		return fmt.Sprintf("The value model still sees %d upside on the board.", analytics.ValueScore)
	default:
		return fmt.Sprintf("Balanced value, crowd heat, and roster fit for %s.", player.Position)
	}
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func countDraftedPlayers(players []models.Player) int {
	totalDrafted := 0
	for _, player := range players {
		if player.Drafted {
			totalDrafted++
		}
	}
	return totalDrafted
}

func expectedTeamForPick(teams []models.Team, mode models.DraftMode, zeroBasedPick int) *models.Team {
	if len(teams) == 0 {
		return nil
	}

	teamIndex := teamIndexForPick(models.NormalizeDraftMode(mode), zeroBasedPick, len(teams))
	if teamIndex < 0 || teamIndex >= len(teams) {
		return nil
	}

	return &teams[teamIndex]
}

func teamIndexForPick(mode models.DraftMode, zeroBasedPick, teamCount int) int {
	if teamCount == 0 {
		return -1
	}

	round := zeroBasedPick / teamCount
	pickInRound := zeroBasedPick % teamCount

	switch mode {
	case models.DraftModeReverseSnake:
		if round%2 == 0 {
			return teamCount - 1 - pickInRound
		}
		return pickInRound
	case models.DraftModeWheel:
		return wheelOrder(teamCount, round)[pickInRound]
	default:
		if round%2 == 0 {
			return pickInRound
		}
		return teamCount - 1 - pickInRound
	}
}

func buildDraftOrder(teams []models.Team, mode models.DraftMode, totalDrafted, totalPlayers int) []models.DraftOrderEntry {
	if len(teams) == 0 || totalPlayers == 0 {
		return nil
	}

	limit := totalPlayers
	if limit > totalDrafted+12 {
		limit = totalDrafted + 12
	}

	entries := make([]models.DraftOrderEntry, 0, limit)
	for zeroBasedPick := 0; zeroBasedPick < limit; zeroBasedPick++ {
		team := expectedTeamForPick(teams, mode, zeroBasedPick)
		if team == nil {
			continue
		}

		entries = append(entries, models.DraftOrderEntry{
			Pick:      zeroBasedPick + 1,
			Round:     (zeroBasedPick / len(teams)) + 1,
			TeamID:    team.ID,
			TeamName:  team.Name,
			Mascot:    team.Mascot,
			Active:    zeroBasedPick == totalDrafted,
			Completed: zeroBasedPick < totalDrafted,
		})
	}

	return entries
}

func buildBingoBoard(totalDrafted, totalPlayers int) []models.BingoSquare {
	promptCount := bingoPromptCount(totalPlayers)
	if promptCount == 0 {
		return nil
	}

	cyclePick := totalDrafted % promptCount
	squares := make([]models.BingoSquare, 0, promptCount)
	for i := 0; i < promptCount; i++ {
		squares = append(squares, models.BingoSquare{
			Index:     i + 1,
			Text:      bingoPrompts[i],
			Active:    i == cyclePick,
			Completed: i < cyclePick,
		})
	}
	return squares
}

func currentBingoPrompt(totalDrafted, totalPlayers int) string {
	promptCount := bingoPromptCount(totalPlayers)
	if promptCount == 0 {
		return ""
	}
	return bingoPrompts[totalDrafted%promptCount]
}

func bingoPromptCount(totalPlayers int) int {
	if totalPlayers <= 0 {
		return 0
	}
	if totalPlayers < len(bingoPrompts) {
		return totalPlayers
	}
	return len(bingoPrompts)
}

func buildWheelSlots(teams []models.Team, totalDrafted, totalPlayers int) []models.WheelSlot {
	if len(teams) == 0 {
		return nil
	}

	round := totalDrafted / len(teams)
	pickInRound := totalDrafted % len(teams)
	picksInRound := len(teams)
	remainingPicks := totalPlayers - (round * len(teams))
	if remainingPicks > 0 && remainingPicks < picksInRound {
		picksInRound = remainingPicks
	}
	if picksInRound <= 0 {
		return nil
	}

	order := wheelOrder(len(teams), round)
	slots := make([]models.WheelSlot, 0, picksInRound)
	for slotIndex, teamIndex := range order[:picksInRound] {
		team := teams[teamIndex]
		slots = append(slots, models.WheelSlot{
			TeamID:   team.ID,
			TeamName: team.Name,
			Mascot:   team.Mascot,
			Active:   slotIndex == pickInRound,
		})
	}
	return slots
}

func wheelOrder(teamCount, round int) []int {
	order := make([]int, teamCount)
	for i := range order {
		order[i] = i
	}

	sort.SliceStable(order, func(i, j int) bool {
		return wheelScore(round, order[i]) < wheelScore(round, order[j])
	})

	return order
}

func wheelScore(round, teamIndex int) uint64 {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "jellycat:%d:%d", round, teamIndex)
	return h.Sum64()
}

func validateTeamTurn(teams []models.Team, mode models.DraftMode, players []models.Player, teamID string) error {
	if len(teams) == 0 {
		return fmt.Errorf("no teams are available")
	}

	totalDrafted := countDraftedPlayers(players)
	if totalDrafted >= len(players) {
		return fmt.Errorf("draft is complete")
	}

	expectedTeam := expectedTeamForPick(teams, mode, totalDrafted)
	if expectedTeam == nil {
		return fmt.Errorf("could not determine current team")
	}

	if expectedTeam.ID != teamID {
		return fmt.Errorf("it is %s's turn", expectedTeam.Name)
	}

	return nil
}
