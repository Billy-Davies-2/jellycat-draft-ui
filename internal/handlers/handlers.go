package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// Publish draft pick event
	h.pubsub.Publish(pubsub.Event{
		Type: "draft:pick",
		Payload: map[string]interface{}{
			"playerId": req.PlayerID,
			"teamId":   req.TeamID,
		},
	})

	// Publish chat event for the system message that was created
	h.pubsub.Publish(pubsub.Event{
		Type: "chat:add",
		Payload: map[string]interface{}{
			"type": "system",
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

	var name, owner, mascot, color string

	// Check content type - handle both JSON and form data
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
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
		name, owner, mascot, color = req.Name, req.Owner, req.Mascot, req.Color
	} else {
		// Handle form data (from htmx forms)
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		name = r.FormValue("name")
		owner = r.FormValue("owner")
		mascot = r.FormValue("mascot")
		color = r.FormValue("color")
	}

	if name == "" {
		http.Error(w, "Team name is required", http.StatusBadRequest)
		return
	}

	team, err := h.dal.AddTeam(name, owner, mascot, color)
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

// UpdateTeam updates an existing team
func (h *APIHandlers) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Owner  string `json:"owner"`
		Mascot string `json:"mascot"`
		Color  string `json:"color"`
	}

	// Check content type - handle both JSON and form data
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		// Handle form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.ID = r.FormValue("id")
		req.Name = r.FormValue("name")
		req.Owner = r.FormValue("owner")
		req.Mascot = r.FormValue("mascot")
		req.Color = r.FormValue("color")
	}

	if req.ID == "" {
		http.Error(w, "Team ID is required", http.StatusBadRequest)
		return
	}

	team, err := h.dal.UpdateTeam(req.ID, req.Name, req.Owner, req.Mascot, req.Color)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "teams:update",
		Payload: map[string]interface{}{
			"id": team.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// DeleteTeam deletes a team
func (h *APIHandlers) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "Team ID is required", http.StatusBadRequest)
		return
	}

	if err := h.dal.DeleteTeam(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "teams:delete",
		Payload: map[string]interface{}{
			"id": req.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
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

// UpdatePlayer updates an existing player
func (h *APIHandlers) UpdatePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var player models.Player
	if err := json.NewDecoder(r.Body).Decode(&player); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if player.ID == "" {
		http.Error(w, "player ID is required", http.StatusBadRequest)
		return
	}

	result, err := h.dal.UpdatePlayer(&player)
	if err != nil {
		logger.Error("Failed to update player", "error", err, "player_id", player.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "players:update",
		Payload: map[string]interface{}{
			"id": result.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// DeletePlayer deletes an existing player
func (h *APIHandlers) DeletePlayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "player ID is required", http.StatusBadRequest)
		return
	}

	err := h.dal.DeletePlayer(req.ID)
	if err != nil {
		logger.Error("Failed to delete player", "error", err, "player_id", req.ID)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.pubsub.Publish(pubsub.Event{
		Type: "players:delete",
		Payload: map[string]interface{}{
			"id": req.ID,
		},
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
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
	logger.Info("SSE client connected", "remoteAddr", r.RemoteAddr)

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to events
	logger.Debug("SSE: Subscribing to pubsub")
	eventChan := h.pubsub.Subscribe()
	defer h.pubsub.Unsubscribe(eventChan)
	logger.Debug("SSE: Subscribed successfully")

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

// UploadImage handles image file uploads for Jellycat pictures
func (h *APIHandlers) UploadImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form with max 10MB file size
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		logger.Error("Failed to parse multipart form", "error", err)
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, header, err := r.FormFile("image")
	if err != nil {
		logger.Error("Failed to get file from form", "error", err)
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true}
	if !allowedExts[ext] {
		http.Error(w, "Invalid file type. Allowed: jpg, jpeg, png, gif, webp", http.StatusBadRequest)
		return
	}

	// Create images directory if it doesn't exist
	imagesDir := "static/images"
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		logger.Error("Failed to create images directory", "error", err)
		http.Error(w, "Failed to create directory", http.StatusInternalServerError)
		return
	}

	// Generate a safe filename (use original name but sanitize it)
	safeFilename := sanitizeFilename(header.Filename)
	destPath := filepath.Join(imagesDir, safeFilename)

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		logger.Error("Failed to create destination file", "error", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer destFile.Close()

	// Copy file contents
	if _, err := io.Copy(destFile, file); err != nil {
		logger.Error("Failed to copy file contents", "error", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	logger.Info("Image uploaded successfully", "filename", safeFilename, "size", header.Size)

	// Return the URL path to the uploaded image
	imageURL := "/static/images/" + safeFilename
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":      imageURL,
		"filename": safeFilename,
	})
}

// ListImages returns a list of all images in the static/images directory
func (h *APIHandlers) ListImages(w http.ResponseWriter, r *http.Request) {
	imagesDir := "static/images"

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty list if directory doesn't exist
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]string{})
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".webp" {
			images = append(images, "/static/images/"+entry.Name())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

// sanitizeFilename removes or replaces characters that could be problematic in filenames
func sanitizeFilename(filename string) string {
	// Replace spaces with hyphens
	filename = strings.ReplaceAll(filename, " ", "-")

	// Keep only alphanumeric, hyphens, underscores, and dots
	var result strings.Builder
	for _, r := range filename {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			result.WriteRune(r)
		}
	}

	return result.String()
}
