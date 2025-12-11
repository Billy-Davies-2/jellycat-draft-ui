package dal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

// MemoryDAL implements DraftDAL using in-memory storage
type MemoryDAL struct {
	mu            sync.RWMutex
	players       []models.Player
	teams         []models.Team
	chat          []models.ChatMessage
	reactionUsers map[string]map[string]map[string]bool // messageID -> emote -> userID -> bool
}

// NewMemoryDAL creates a new in-memory data access layer
func NewMemoryDAL() *MemoryDAL {
	dal := &MemoryDAL{
		players:       getDefaultPlayers(),
		teams:         getDefaultTeams(),
		chat:          []models.ChatMessage{},
		reactionUsers: make(map[string]map[string]map[string]bool),
	}

	// Add welcome messages
	dal.AddChatMessage("Welcome to the Jellycat Draft! 🎉", "system")
	dal.AddChatMessage("Tip: Click a Jellycat card to draft it!", "system")
	dal.AddChatMessage("Who will snag Bashful Bunny first? 🐰", "system")

	return dal
}

func (m *MemoryDAL) GetState() (*models.DraftState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create copies to avoid race conditions
	state := &models.DraftState{
		Players: make([]models.Player, len(m.players)),
		Teams:   make([]models.Team, len(m.teams)),
		Chat:    make([]models.ChatMessage, len(m.chat)),
	}

	copy(state.Players, m.players)
	copy(state.Teams, m.teams)
	copy(state.Chat, m.chat)

	return state, nil
}

func (m *MemoryDAL) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players = getDefaultPlayers()
	m.teams = getDefaultTeams()
	m.chat = []models.ChatMessage{}
	m.reactionUsers = make(map[string]map[string]map[string]bool)

	return nil
}

func (m *MemoryDAL) AddPlayer(player *models.Player) (*models.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if player.ID == "" {
		player.ID = genID("player")
	}

	m.players = append(m.players, *player)
	return player, nil
}

func (m *MemoryDAL) SetPlayerPoints(id string, points int) (*models.Player, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.players {
		if m.players[i].ID == id {
			m.players[i].Points = points

			// Update in team rosters too
			for j := range m.teams {
				for k := range m.teams[j].Players {
					if m.teams[j].Players[k].ID == id {
						m.teams[j].Players[k].Points = points
					}
				}
			}

			return &m.players[i], nil
		}
	}

	return nil, fmt.Errorf("player not found")
}

func (m *MemoryDAL) ReorderTeams(order []string) ([]models.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idToTeam := make(map[string]models.Team)
	for _, team := range m.teams {
		idToTeam[team.ID] = team
	}

	reordered := []models.Team{}
	for _, id := range order {
		if team, ok := idToTeam[id]; ok {
			reordered = append(reordered, team)
			delete(idToTeam, id)
		}
	}

	// Append any missing teams
	for _, team := range m.teams {
		if _, ok := idToTeam[team.ID]; ok {
			reordered = append(reordered, team)
		}
	}

	m.teams = reordered
	return m.teams, nil
}

func (m *MemoryDAL) DraftPlayer(playerID, teamID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var player *models.Player
	var team *models.Team

	for i := range m.players {
		if m.players[i].ID == playerID {
			player = &m.players[i]
			break
		}
	}

	for i := range m.teams {
		if m.teams[i].ID == teamID {
			team = &m.teams[i]
			break
		}
	}

	if player == nil {
		return fmt.Errorf("player not found")
	}
	if team == nil {
		return fmt.Errorf("team not found")
	}
	if player.Drafted {
		return fmt.Errorf("player already drafted")
	}

	player.Drafted = true
	player.DraftedBy = team.Name
	team.Players = append(team.Players, *player)

	// Add system message
	msg := fmt.Sprintf("%s %s drafted %s (%s • %s)", team.Mascot, team.Name, player.Name, player.Team, player.Position)
	m.addChatMessageUnsafe(msg, "system")

	return nil
}

func (m *MemoryDAL) AddChatMessage(text, msgType string) (*models.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.addChatMessageUnsafe(text, msgType), nil
}

func (m *MemoryDAL) addChatMessageUnsafe(text, msgType string) *models.ChatMessage {
	msg := &models.ChatMessage{
		ID:     genID("msg"),
		TS:     time.Now().UnixMilli(),
		Type:   msgType,
		Text:   text,
		Emotes: make(map[string]int),
	}
	m.chat = append(m.chat, *msg)
	return msg
}

func (m *MemoryDAL) AddReaction(messageID, emote, userID string) (*models.ChatMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var msg *models.ChatMessage
	for i := range m.chat {
		if m.chat[i].ID == messageID {
			msg = &m.chat[i]
			break
		}
	}

	if msg == nil {
		return nil, fmt.Errorf("message not found")
	}

	uid := userID
	if uid == "" {
		uid = "anon"
	}

	// Check if user already reacted with this emote
	if m.reactionUsers[messageID] == nil {
		m.reactionUsers[messageID] = make(map[string]map[string]bool)
	}
	if m.reactionUsers[messageID][emote] == nil {
		m.reactionUsers[messageID][emote] = make(map[string]bool)
	}

	if m.reactionUsers[messageID][emote][uid] {
		return msg, nil // Already reacted
	}

	m.reactionUsers[messageID][emote][uid] = true
	msg.Emotes[emote]++

	return msg, nil
}

func (m *MemoryDAL) AddTeam(name, owner, mascot, color string) (*models.Team, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mascots := []string{"🦊", "🐻", "🐰", "🐱", "🐑", "🦒", "🐨", "🦁", "🐼", "🦄", "🐯", "🐶"}
	colors := []string{
		"bg-orange-100 border-orange-300",
		"bg-amber-100 border-amber-300",
		"bg-pink-100 border-pink-300",
		"bg-purple-100 border-purple-300",
		"bg-blue-100 border-blue-300",
		"bg-yellow-100 border-yellow-300",
		"bg-green-100 border-green-300",
	}

	if owner == "" {
		owner = "Anonymous"
	}
	if mascot == "" {
		mascot = mascots[len(m.teams)%len(mascots)]
	}
	if color == "" {
		color = colors[len(m.teams)%len(colors)]
	}

	team := &models.Team{
		ID:      genID("team"),
		Name:    name,
		Owner:   owner,
		Mascot:  mascot,
		Color:   color,
		Players: []models.Player{},
	}

	m.teams = append(m.teams, *team)

	// Add system message
	msg := fmt.Sprintf("New team joined the draft: %s %s (Owner: %s)", team.Mascot, team.Name, team.Owner)
	m.addChatMessageUnsafe(msg, "system")

	return team, nil
}

func genID(prefix string) string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func getDefaultPlayers() []models.Player {
	return []models.Player{
		{ID: "1", Name: "Bashful Bunny", Position: "CC", Team: "Woodland", Points: 324, CuddlePoints: 50, Tier: models.TierS, Drafted: false, Image: "/images/bashful-bunny.png"},
		{ID: "2", Name: "Fuddlewuddle Lion", Position: "SS", Team: "Safari", Points: 298, CuddlePoints: 50, Tier: models.TierS, Drafted: false, Image: "/images/fuddlewuddle-lion.png"},
		{ID: "3", Name: "Cordy Roy Elephant", Position: "HH", Team: "Safari", Points: 287, CuddlePoints: 50, Tier: models.TierS, Drafted: false, Image: "/images/cordy-roy-elephant.png"},
		{ID: "4", Name: "Blossom Tulip Bunny", Position: "CH", Team: "Garden", Points: 251, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/blossom-tulip-bunny.png"},
		{ID: "5", Name: "Amuseable Avocado", Position: "CC", Team: "Kitchen", Points: 312, CuddlePoints: 50, Tier: models.TierS, Drafted: false, Image: "/images/amuseable-avocado.png"},
		{ID: "6", Name: "Octopus Ollie", Position: "SS", Team: "Ocean", Points: 276, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/octopus-ollie.png"},
		{ID: "7", Name: "Jellycat Dragon", Position: "HH", Team: "Fantasy", Points: 268, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/jellycat-dragon.png"},
		{ID: "8", Name: "Bashful Lamb", Position: "CH", Team: "Farm", Points: 245, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/bashful-lamb.png"},
		{ID: "9", Name: "Amuseable Pineapple", Position: "CC", Team: "Tropical", Points: 289, CuddlePoints: 50, Tier: models.TierS, Drafted: false, Image: "/images/amuseable-pineapple.png"},
		{ID: "10", Name: "Cordy Roy Fox", Position: "SS", Team: "Woodland", Points: 234, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/cordy-roy-fox.png"},
		{ID: "11", Name: "Blossom Peach Bunny", Position: "HH", Team: "Garden", Points: 256, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/blossom-peach-bunny.png"},
		{ID: "12", Name: "Amuseable Taco", Position: "CH", Team: "Kitchen", Points: 267, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/amuseable-taco.png"},
		{ID: "13", Name: "Bashful Unicorn", Position: "CC", Team: "Fantasy", Points: 278, CuddlePoints: 50, Tier: models.TierA, Drafted: false, Image: "/images/bashful-unicorn.png"},
		{ID: "14", Name: "Jellycat Penguin", Position: "SS", Team: "Arctic", Points: 243, CuddlePoints: 50, Tier: models.TierB, Drafted: false, Image: "/images/jellycat-penguin.png"},
		{ID: "15", Name: "Amuseable Moon", Position: "HH", Team: "Space", Points: 229, CuddlePoints: 50, Tier: models.TierB, Drafted: false, Image: "/images/amuseable-moon.png"},
		{ID: "16", Name: "Cordy Roy Pig", Position: "CH", Team: "Farm", Points: 241, CuddlePoints: 50, Tier: models.TierB, Drafted: false, Image: "/images/cordy-roy-pig.png"},
		{ID: "17", Name: "Bashful Tiger", Position: "SS", Team: "Safari", Points: 235, CuddlePoints: 50, Tier: models.TierB, Drafted: false, Image: "/images/bashful-tiger.png"},
		{ID: "18", Name: "Amuseable Donut", Position: "CC", Team: "Kitchen", Points: 228, CuddlePoints: 50, Tier: models.TierB, Drafted: false, Image: "/images/amuseable-donut.png"},
	}
}

func getDefaultTeams() []models.Team {
	return []models.Team{
		{ID: "1", Name: "Fluffy Foxes", Owner: "Sarah", Mascot: "🦊", Color: "bg-orange-100 border-orange-300", Players: []models.Player{}},
		{ID: "2", Name: "Cuddly Bears", Owner: "Mike", Mascot: "🐻", Color: "bg-amber-100 border-amber-300", Players: []models.Player{}},
		{ID: "3", Name: "Snuggly Bunnies", Owner: "Emma", Mascot: "🐰", Color: "bg-pink-100 border-pink-300", Players: []models.Player{}},
		{ID: "4", Name: "Cozy Cats", Owner: "Alex", Mascot: "🐱", Color: "bg-purple-100 border-purple-300", Players: []models.Player{}},
		{ID: "5", Name: "Soft Sheep", Owner: "Jordan", Mascot: "🐑", Color: "bg-blue-100 border-blue-300", Players: []models.Player{}},
		{ID: "6", Name: "Gentle Giraffes", Owner: "Taylor", Mascot: "🦒", Color: "bg-yellow-100 border-yellow-300", Players: []models.Player{}},
	}
}
