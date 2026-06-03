package models

// Tier represents the player tier rating
type Tier string

const (
	TierS Tier = "S"
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
)

// DraftMode controls how teams are selected for each pick.
type DraftMode string

const (
	DraftModeStandard     DraftMode = "standard"
	DraftModeReverseSnake DraftMode = "reverse-snake"
	DraftModeBingo        DraftMode = "bingo"
	DraftModeWheel        DraftMode = "wheel"
)

// DraftSettings stores the active draft experience.
type DraftSettings struct {
	Mode        DraftMode `json:"mode"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
}

// DraftModeOption is used by the admin UI to present supported modes.
type DraftModeOption struct {
	Mode        DraftMode `json:"mode"`
	Name        string    `json:"name"`
	Tagline     string    `json:"tagline"`
	Description string    `json:"description"`
}

// DraftOrderEntry describes a pick in the visible order preview.
type DraftOrderEntry struct {
	Pick      int    `json:"pick"`
	Round     int    `json:"round"`
	TeamID    string `json:"teamId"`
	TeamName  string `json:"teamName"`
	Mascot    string `json:"mascot"`
	Active    bool   `json:"active"`
	Completed bool   `json:"completed"`
}

// BingoSquare represents one prompt on the bingo draft board.
type BingoSquare struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	Active    bool   `json:"active"`
	Completed bool   `json:"completed"`
}

// WheelSlot represents one team on the spin wheel.
type WheelSlot struct {
	TeamID   string `json:"teamId"`
	TeamName string `json:"teamName"`
	Mascot   string `json:"mascot"`
	Active   bool   `json:"active"`
}

// PlayerAnalytics contains display-ready scouting signals for a player.
type PlayerAnalytics struct {
	PickScore  int    `json:"pickScore"`
	ValueScore int    `json:"valueScore"`
	CrowdHeat  int    `json:"crowdHeat"`
	NeedFit    int    `json:"needFit"`
	TrendDelta int    `json:"trendDelta"`
	TrendLabel string `json:"trendLabel"`
	Label      string `json:"label"`
	Reason     string `json:"reason"`
	Sparkline  []int  `json:"sparkline"`
	Suggested  bool   `json:"suggested"`
}

// DraftRecommendation is the current best available pick for the active team.
type DraftRecommendation struct {
	Player     Player `json:"player"`
	Reason     string `json:"reason"`
	Confidence int    `json:"confidence"`
	ModelLabel string `json:"modelLabel"`
}

// DraftModeOptions returns the supported draft experiences.
func DraftModeOptions() []DraftModeOption {
	return []DraftModeOption{
		{
			Mode:        DraftModeStandard,
			Name:        "Standard Fantasy",
			Tagline:     "Classic serpentine draft",
			Description: "Round one follows the team list, then each round reverses order.",
		},
		{
			Mode:        DraftModeReverseSnake,
			Name:        "Reverse Snake",
			Tagline:     "The last team strikes first",
			Description: "Round one starts from the bottom of the list, then alternates back and forth.",
		},
		{
			Mode:        DraftModeBingo,
			Name:        "Bingo Draft",
			Tagline:     "Every pick has a prompt",
			Description: "Uses standard fantasy order while each pick lights up a bingo challenge.",
		},
		{
			Mode:        DraftModeWheel,
			Name:        "Spin the Wheel",
			Tagline:     "A shuffled wheel chooses the table",
			Description: "Each round uses a deterministic wheel shuffle so the current pick stays stable.",
		},
	}
}

// DefaultDraftSettings returns the production-safe default mode.
func DefaultDraftSettings() DraftSettings {
	return DraftSettingsForMode(DraftModeStandard)
}

// DraftSettingsForMode returns display metadata for a normalized mode.
func DraftSettingsForMode(mode DraftMode) DraftSettings {
	switch NormalizeDraftMode(mode) {
	case DraftModeReverseSnake:
		return DraftSettings{Mode: DraftModeReverseSnake, Name: "Reverse Snake", Description: "Round one starts with the last team, then alternates every round."}
	case DraftModeBingo:
		return DraftSettings{Mode: DraftModeBingo, Name: "Bingo Draft", Description: "Standard fantasy order with a rotating bingo prompt for each pick."}
	case DraftModeWheel:
		return DraftSettings{Mode: DraftModeWheel, Name: "Spin the Wheel", Description: "A stable shuffled wheel picks the team order for each round."}
	default:
		return DraftSettings{Mode: DraftModeStandard, Name: "Standard Fantasy", Description: "Classic fantasy serpentine order."}
	}
}

// NormalizeDraftMode maps unknown or empty values to the default draft mode.
func NormalizeDraftMode(mode DraftMode) DraftMode {
	switch mode {
	case DraftModeStandard, DraftModeReverseSnake, DraftModeBingo, DraftModeWheel:
		return mode
	default:
		return DraftModeStandard
	}
}

// Player represents a Jellycat player
type Player struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Position     string          `json:"position"`
	Team         string          `json:"team"`
	Points       int             `json:"points"`
	CuddlePoints int             `json:"cuddlePoints"`
	Tier         Tier            `json:"tier"`
	Drafted      bool            `json:"drafted"`
	DraftedBy    string          `json:"draftedBy,omitempty"`
	Image        string          `json:"image"`
	Analytics    PlayerAnalytics `json:"analytics"`
}

// Team represents a draft team
type Team struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Owner   string   `json:"owner"`
	Mascot  string   `json:"mascot"`
	Color   string   `json:"color"`
	Players []Player `json:"players"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID     string         `json:"id"`
	TS     int64          `json:"ts"`
	Type   string         `json:"type"` // "system" or "user"
	Text   string         `json:"text"`
	Emotes map[string]int `json:"emotes"`
}

// DraftState represents the complete state of the draft
type DraftState struct {
	Players             []Player             `json:"players"`
	Teams               []Team               `json:"teams"`
	Chat                []ChatMessage        `json:"chat"`
	Settings            DraftSettings        `json:"settings"`
	CurrentPick         int                  `json:"currentPick"`
	CurrentRound        int                  `json:"currentRound"`
	PickInRound         int                  `json:"pickInRound"`
	CurrentTeamID       string               `json:"currentTeamId"`
	CurrentTeamName     string               `json:"currentTeamName"`
	DraftOrder          []DraftOrderEntry    `json:"draftOrder"`
	BingoBoard          []BingoSquare        `json:"bingoBoard,omitempty"`
	CurrentBingoPrompt  string               `json:"currentBingoPrompt,omitempty"`
	WheelSlots          []WheelSlot          `json:"wheelSlots,omitempty"`
	SuggestedPick       *DraftRecommendation `json:"suggestedPick,omitempty"`
	AnalyticsConfigured bool                 `json:"analyticsConfigured"`
}

// PlayerProfile represents extended player information
type PlayerProfile struct {
	Player
	Metrics struct {
		Consistency int     `json:"consistency"`
		Popularity  int     `json:"popularity"`
		Efficiency  int     `json:"efficiency"`
		TrendDelta  float64 `json:"trendDelta"`
	} `json:"metrics"`
}
