package models

// Tier represents the player tier rating
type Tier string

const (
	TierS Tier = "S"
	TierA Tier = "A"
	TierB Tier = "B"
	TierC Tier = "C"
)

// Player represents a Jellycat player
type Player struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Position     string `json:"position"`
	Team         string `json:"team"`
	Points       int    `json:"points"`
	CuddlePoints int    `json:"cuddlePoints"`
	Tier         Tier   `json:"tier"`
	Drafted      bool   `json:"drafted"`
	DraftedBy    string `json:"draftedBy,omitempty"`
	Image        string `json:"image"`
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
	Players []Player      `json:"players"`
	Teams   []Team        `json:"teams"`
	Chat    []ChatMessage `json:"chat"`
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
