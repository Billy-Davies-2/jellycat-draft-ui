package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
)

// APIHandlers contains all API handler methods
type APIHandlers struct {
	dal    dal.DraftDAL
	pubsub *pubsub.PubSub
}

// NewAPIHandlers creates a new API handlers instance
func NewAPIHandlers(dal dal.DraftDAL, ps *pubsub.PubSub) *APIHandlers {
	return &APIHandlers{
		dal:    dal,
		pubsub: ps,
	}
}

// GetDraftState returns the current draft state
func (h *APIHandlers) GetDraftState(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Getting draft state")
	state, err := h.dal.GetState()
	if err != nil {
		logger.Error("Failed to get draft state", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// DraftPick handles player draft selection
func (h *APIHandlers) DraftPick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PlayerID string `json:"playerId"`
		TeamID   string `json:"teamId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Warn("Failed to decode draft pick request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Info("Drafting player", "player_id", req.PlayerID, "team_id", req.TeamID)
	if err := h.dal.DraftPlayer(req.PlayerID, req.TeamID); err != nil {
		logger.Error("Failed to draft player", "error", err, "player_id", req.PlayerID, "team_id", req.TeamID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Publish event
	h.pubsub.Publish(pubsub.Event{
		Type: "draft:pick",
		Payload: map[string]interface{}{
			"playerId": req.PlayerID,
			"teamId":   req.TeamID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ResetDraft resets the draft to initial state
func (h *APIHandlers) ResetDraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Info("Resetting draft")
	if err := h.dal.Reset(); err != nil {
		logger.Error("Failed to reset draft", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{Type: "draft:reset"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// ListTeams returns all teams
func (h *APIHandlers) ListTeams(w http.ResponseWriter, r *http.Request) {
	state, err := h.dal.GetState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state.Teams)
}

// AddTeam creates a new team
func (h *APIHandlers) AddTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Owner  string `json:"owner"`
		Mascot string `json:"mascot"`
		Color  string `json:"color"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	team, err := h.dal.AddTeam(req.Name, req.Owner, req.Mascot, req.Color)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "teams:add",
		Payload: map[string]interface{}{
			"id": team.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// ReorderTeams reorders the team draft order
func (h *APIHandlers) ReorderTeams(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Order []string `json:"order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	teams, err := h.dal.ReorderTeams(req.Order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{Type: "teams:reorder"})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// AddPlayer adds a new player
func (h *APIHandlers) AddPlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var player models.Player
	if err := json.NewDecoder(r.Body).Decode(&player); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.dal.AddPlayer(&player)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "players:add",
		Payload: map[string]interface{}{
			"id": result.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SetPlayerPoints updates a player's points
func (h *APIHandlers) SetPlayerPoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Points int    `json:"points"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	player, err := h.dal.SetPlayerPoints(req.ID, req.Points)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "players:updatePoints",
		Payload: map[string]interface{}{
			"id":     player.ID,
			"points": player.Points,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

// GetPlayerProfile returns extended player information
func (h *APIHandlers) GetPlayerProfile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing id parameter", http.StatusBadRequest)
		return
	}

	state, err := h.dal.GetState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var player *models.Player
	for _, p := range state.Players {
		if p.ID == id {
			player = &p
			break
		}
	}

	if player == nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Generate mock metrics
	profile := models.PlayerProfile{Player: *player}
	seed := player.Points
	for _, c := range player.ID {
		seed += int(c)
	}

	norm := func(x int) int {
		return int(math.Max(0, math.Min(100, float64(x))))
	}

	profile.Metrics.Consistency = norm((seed * 13) % 101)
	profile.Metrics.Popularity = norm((seed * 29) % 101)
	profile.Metrics.Efficiency = norm((seed * 47) % 101)
	profile.Metrics.TrendDelta = float64(((seed%15)-7)/7.0) * 100 / 100

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// ListChat returns all chat messages
func (h *APIHandlers) ListChat(w http.ResponseWriter, r *http.Request) {
	state, err := h.dal.GetState()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state.Chat)
}

// SendChatMessage sends a new chat message
func (h *APIHandlers) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text string `json:"text"`
		Type string `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "user"
	}

	msg, err := h.dal.AddChatMessage(req.Text, req.Type)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "chat:add",
		Payload: map[string]interface{}{
			"id": msg.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

// AddReaction adds a reaction to a chat message
func (h *APIHandlers) AddReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MessageID string `json:"messageId"`
		Emote     string `json:"emote"`
		User      string `json:"user"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msg, err := h.dal.AddReaction(req.MessageID, req.Emote, req.User)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "chat:react",
		Payload: map[string]interface{}{
			"id":    msg.ID,
			"emote": req.Emote,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

// EventsSSE provides Server-Sent Events for realtime updates
func (h *APIHandlers) EventsSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to events
	eventChan := h.pubsub.Subscribe()
	defer h.pubsub.Unsubscribe(eventChan)

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Listen for events
	for {
		select {
		case event := <-eventChan:
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			logger.Debug("SSE client disconnected")
			return
		case <-time.After(30 * time.Second):
			// Send keepalive ping
			fmt.Fprintf(w, ": keepalive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
