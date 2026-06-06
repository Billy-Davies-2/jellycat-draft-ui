package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
	qrcode "github.com/skip2/go-qrcode"
)

const roomCodeLength = 4

var draftRoom *roomState

type roomState struct {
	code string
}

type roomJoinRequest struct {
	Code     string `json:"code"`
	Username string `json:"username"`
	TeamName string `json:"teamName"`
	TeamID   string `json:"teamId"`
}

type roomJoinResponse struct {
	OK   bool        `json:"ok"`
	Code string      `json:"code"`
	Team models.Team `json:"team"`
}

func newRoomState(configuredCode string) *roomState {
	code := normalizeRoomCode(configuredCode)
	if !isValidRoomCode(code) {
		code = generateRoomCode()
	}
	return &roomState{code: code}
}

func (room *roomState) Code() string {
	if room == nil {
		return ""
	}
	return room.code
}

func (room *roomState) Matches(code string) bool {
	return room != nil && normalizeRoomCode(code) == room.code
}

func generateRoomCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	const digits = "23456789"
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ"

	for {
		code := make([]byte, roomCodeLength)
		for index := range code {
			code[index] = alphabet[randomInt(len(alphabet))]
		}

		value := string(code)
		if strings.ContainsAny(value, digits) && strings.ContainsAny(value, letters) {
			return value
		}
	}
}

func randomInt(max int) int {
	value, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(fmt.Sprintf("failed to generate room code: %v", err))
	}
	return int(value.Int64())
}

func normalizeRoomCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "")
	code = strings.ReplaceAll(code, "-", "")
	return code
}

func isValidRoomCode(code string) bool {
	if len(code) != roomCodeLength {
		return false
	}

	hasLetter := false
	hasDigit := false
	for _, char := range code {
		switch {
		case char >= 'A' && char <= 'Z':
			hasLetter = true
		case char >= '0' && char <= '9':
			hasDigit = true
		default:
			return false
		}
	}
	return hasLetter && hasDigit
}

func roomInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":     draftRoom.Code(),
		"joinPath": joinPath(),
		"joinUrl":  joinURL(r),
		"qrPath":   joinQRPath(),
	})
}

func roomQRHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if code := normalizeRoomCode(r.URL.Query().Get("code")); code != "" && !draftRoom.Matches(code) {
		http.Error(w, "Invalid room code", http.StatusUnauthorized)
		return
	}

	png, err := qrcode.Encode(joinURL(r), qrcode.Medium, 512)
	if err != nil {
		http.Error(w, "Failed to generate QR code", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func roomJoinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	request, err := decodeRoomJoinRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !draftRoom.Matches(request.Code) {
		http.Error(w, "Invalid room code", http.StatusUnauthorized)
		return
	}

	request.Username = cleanRoomValue(request.Username, 80)
	request.TeamName = cleanRoomValue(request.TeamName, 80)
	request.TeamID = cleanRoomValue(request.TeamID, 120)

	if request.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	var team *models.Team
	if request.TeamID != "" {
		team, err = findTeamByID(request.TeamID)
		if err == nil && strings.TrimSpace(team.Owner) == "" {
			team, err = dataStore.UpdateTeam(team.ID, team.Name, request.Username, team.Mascot, team.Color)
			if err == nil {
				publishRoomTeamUpdateEvent(team)
			}
		}
	} else {
		if request.TeamName == "" {
			request.TeamName = request.Username
		}
		team, err = dataStore.AddTeam(request.TeamName, request.Username, "", "")
		if err == nil {
			publishRoomJoinEvents(team)
		}
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(roomJoinResponse{OK: true, Code: draftRoom.Code(), Team: *team})
}

func decodeRoomJoinRequest(r *http.Request) (roomJoinRequest, error) {
	var request roomJoinRequest
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return request, err
		}
		return request, nil
	}

	if err := r.ParseForm(); err != nil {
		return request, err
	}
	request.Code = r.FormValue("code")
	request.Username = r.FormValue("username")
	request.TeamName = r.FormValue("teamName")
	request.TeamID = r.FormValue("teamId")
	return request, nil
}

func cleanRoomValue(value string, maxLength int) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "<", "")
	value = strings.ReplaceAll(value, ">", "")
	if len(value) > maxLength {
		value = value[:maxLength]
	}
	return value
}

func findTeamByID(teamID string) (*models.Team, error) {
	state, err := dataStore.GetState()
	if err != nil {
		return nil, err
	}
	for _, team := range state.Teams {
		if team.ID == teamID {
			teamCopy := team
			return &teamCopy, nil
		}
	}
	return nil, fmt.Errorf("team not found")
}

func publishRoomJoinEvents(team *models.Team) {
	if ps == nil || team == nil {
		return
	}
	ps.Publish(pubsub.Event{
		Type: "teams:add",
		Payload: map[string]interface{}{
			"id": team.ID,
		},
	})
	ps.Publish(pubsub.Event{
		Type: "chat:add",
		Payload: map[string]interface{}{
			"type": "system",
		},
	})
}

func publishRoomTeamUpdateEvent(team *models.Team) {
	if ps == nil || team == nil {
		return
	}
	ps.Publish(pubsub.Event{
		Type: "teams:update",
		Payload: map[string]interface{}{
			"id": team.ID,
		},
	})
}

func roomTemplateData(r *http.Request) map[string]string {
	return map[string]string{
		"RoomCode":   draftRoom.Code(),
		"JoinPath":   joinPath(),
		"JoinURL":    joinURL(r),
		"JoinQRPath": joinQRPath(),
	}
}

func joinPath() string {
	return "/join?code=" + url.QueryEscape(draftRoom.Code())
}

func joinQRPath() string {
	return "/api/room/qr?code=" + url.QueryEscape(draftRoom.Code())
}

func joinURL(r *http.Request) string {
	if publicURL := strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"); publicURL != "" {
		return publicURL + joinPath()
	}

	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	return scheme + "://" + r.Host + joinPath()
}
