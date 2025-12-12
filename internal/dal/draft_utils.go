package dal

import "github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"

// CalculateCurrentPick calculates the current pick number and determines whose turn it is
// based on the number of drafted players and teams using snake draft logic.
func CalculateCurrentPick(state *models.DraftState, players []models.Player) {
	// Calculate current pick number
	totalDrafted := 0
	for _, player := range players {
		if player.Drafted {
			totalDrafted++
		}
	}
	state.CurrentPick = totalDrafted + 1

	// Determine current team (snake draft style)
	if len(state.Teams) > 0 {
		teamCount := len(state.Teams)
		round := totalDrafted / teamCount
		pickInRound := totalDrafted % teamCount

		var teamIndex int
		if round%2 == 0 {
			// Even rounds go forward (0, 1, 2, ...)
			teamIndex = pickInRound
		} else {
			// Odd rounds go backward (... 2, 1, 0)
			teamIndex = teamCount - 1 - pickInRound
		}

		if teamIndex < len(state.Teams) {
			state.CurrentTeamID = state.Teams[teamIndex].ID
			state.CurrentTeamName = state.Teams[teamIndex].Name
		}
	}
}
