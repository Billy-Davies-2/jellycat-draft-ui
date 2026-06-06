package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/auth"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

func TestRequireAdminAPIRequiresLogin(t *testing.T) {
	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/players/update", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if called {
		t.Fatal("handler should not be called without a logged-in user")
	}
}

func TestRequireAdminAPIRequiresAdmin(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "email")
	t.Setenv("AUTH_ADMIN_VALUE", "billy.davies.10@icloud.com")

	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	request := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/players/update", nil), &auth.User{Email: "not-admin@example.com"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("handler should not be called for a non-admin user")
	}
}

func TestRequireAdminAPIAllowsAdmin(t *testing.T) {
	t.Setenv("AUTH_ADMIN_CLAIM", "email")
	t.Setenv("AUTH_ADMIN_VALUE", "billy.davies.10@icloud.com")

	called := false
	handler := requireAdminAPI(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})

	request := requestWithUser(httptest.NewRequest(http.MethodPost, "/api/players/update", nil), &auth.User{Email: "billy.davies.10@icloud.com"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusAccepted)
	}
	if !called {
		t.Fatal("handler should be called for the configured admin user")
	}
}

func TestBuildFeaturedProspectsSpreadsCategories(t *testing.T) {
	players := []models.Player{
		{ID: "space-1", Name: "Space One", Team: "Space", Position: "CC", Image: "/images/space-1.png"},
		{ID: "space-2", Name: "Space Two", Team: "Space", Position: "SS", Image: "/images/space-2.png"},
		{ID: "ocean-1", Name: "Ocean One", Team: "Ocean", Position: "HH", Image: "/images/ocean-1.png"},
		{ID: "garden-1", Name: "Garden One", Team: "Garden", Position: "DD", Image: "/images/garden-1.png"},
	}

	prospects := buildFeaturedProspects(players, "ROOM|standard")
	if len(prospects) != 3 {
		t.Fatalf("featured prospects length = %d, want 3", len(prospects))
	}

	categories := map[string]bool{}
	for index, prospect := range prospects {
		categories[prospect.Category] = true
		if prospect.Image == "" || prospect.Name == "" || prospect.Label == "" {
			t.Fatalf("featured prospect missing display fields: %+v", prospect)
		}
		if prospect.FrameIndex != index {
			t.Fatalf("featured prospect frame index = %d, want %d", prospect.FrameIndex, index)
		}
	}

	if len(categories) != 3 {
		t.Fatalf("featured prospects should prefer distinct categories, got %v", categories)
	}
}

func TestBuildFeaturedProspectsIgnoresDraftedPlayers(t *testing.T) {
	players := []models.Player{
		{ID: "drafted", Name: "Drafted", Team: "Space", Drafted: true, Image: "/images/drafted.png"},
		{ID: "available", Name: "Available", Team: "Ocean", Image: "/images/available.png"},
	}

	prospects := buildFeaturedProspects(players, "ROOM|standard")
	if len(prospects) != 1 {
		t.Fatalf("featured prospects length = %d, want 1", len(prospects))
	}
	if prospects[0].Name != "Available" {
		t.Fatalf("featured prospect = %q, want available player", prospects[0].Name)
	}
}

func TestAdminTemplateRendersTeamManagement(t *testing.T) {
	tmpl, err := template.ParseFiles("templates/base.html", "templates/admin.html")
	if err != nil {
		t.Fatalf("parse admin template: %v", err)
	}

	data := map[string]interface{}{
		"Players": []models.Player{},
		"Teams": []models.Team{
			{ID: "team-1", Name: "Test Team", Owner: "", Mascot: "T", Color: "bg-blue-100 border-blue-300", Players: []models.Player{}},
		},
		"Settings":            models.DefaultDraftSettings(),
		"ModeOptions":         models.DraftModeOptions(),
		"AnalyticsConfigured": false,
		"User":                &auth.User{Name: "Admin"},
		"IsAdmin":             true,
	}

	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "base.html", data); err != nil {
		t.Fatalf("execute admin template: %v", err)
	}

	for _, expected := range []string{"Manage Teams", "Add Team", "Move Up", "Unassigned", "editTeamColor"} {
		if !strings.Contains(rendered.String(), expected) {
			t.Fatalf("admin template missing %q", expected)
		}
	}
}

func TestDraftAndPickTemplatesRenderRoomUX(t *testing.T) {
	data := map[string]interface{}{
		"Players": []models.Player{
			{
				ID:           "player-1",
				Name:         "Keli Pelican",
				Position:     "CH",
				Team:         "Ocean Motion",
				CuddlePoints: 73,
				Tier:         models.TierC,
				Image:        "/images/keli.png",
				Analytics: models.PlayerAnalytics{
					PickScore:  81,
					ValueScore: 72,
					CrowdHeat:  80,
					NeedFit:    70,
					TrendLabel: "+2%",
					Label:      "Steady pick",
					Reason:     "Balanced profile.",
					Sparkline:  []int{20, 40, 60},
				},
			},
		},
		"Teams": []models.Team{
			{ID: "team-1", Name: "Draft Slot 1", Owner: "Taylor", Mascot: "T", Color: "bg-blue-100 border-blue-300", Players: []models.Player{}},
		},
		"Chat":                []models.ChatMessage{},
		"Settings":            models.DefaultDraftSettings(),
		"ModeOptions":         models.DraftModeOptions(),
		"IsBingoMode":         false,
		"IsWheelMode":         false,
		"DraftOrder":          []models.DraftOrderEntry{{Pick: 1, Round: 1, TeamID: "team-1", TeamName: "Draft Slot 1", Mascot: "T", Active: true}},
		"BingoBoard":          []models.BingoSquare{},
		"CurrentBingoPrompt":  "",
		"WheelSlots":          []models.WheelSlot{},
		"SuggestedPick":       nil,
		"AnalyticsConfigured": false,
		"User":                &auth.User{Name: "Taylor"},
		"IsAdmin":             false,
		"CurrentPick":         1,
		"CurrentRound":        1,
		"PickInRound":         1,
		"CurrentTeamID":       "team-1",
		"CurrentTeamName":     "Draft Slot 1",
		"UserTeamID":          "team-1",
		"IsUserTurn":          true,
		"InitialRoomCode":     "A123",
		"RoomCode":            "A123",
		"JoinPath":            "/join?code=A123",
		"JoinURL":             "https://example.test/join?code=A123",
		"JoinQRPath":          "/api/room/qr?code=A123",
	}

	expectedByTemplate := map[string][]string{
		"templates/draft.html": {"Draft Lobby", "Waiting Room", "Taylor", "Draft Slot 1"},
		"templates/pick.html":  {"Optional team nickname", "Draft Slots", "Taylor", "Draft Slot 1"},
	}

	for _, templateFile := range []string{"templates/draft.html", "templates/pick.html"} {
		t.Run(templateFile, func(t *testing.T) {
			tmpl, err := template.ParseFiles("templates/base.html", templateFile)
			if err != nil {
				t.Fatalf("parse template: %v", err)
			}

			var rendered bytes.Buffer
			if err := tmpl.ExecuteTemplate(&rendered, "base.html", data); err != nil {
				t.Fatalf("execute template: %v", err)
			}

			output := rendered.String()
			for _, expected := range expectedByTemplate[templateFile] {
				if !strings.Contains(output, expected) {
					t.Fatalf("template missing %q", expected)
				}
			}
		})
	}
}

func TestRoomJoinDefaultsTeamNameToUsername(t *testing.T) {
	originalStore := dataStore
	originalRoom := draftRoom
	originalPubSub := ps
	defer func() {
		dataStore = originalStore
		draftRoom = originalRoom
		ps = originalPubSub
	}()

	dataStore = dal.NewMemoryDAL()
	draftRoom = newRoomState("A123")
	ps = nil

	body := strings.NewReader(`{"code":"A123","username":"Taylor"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/room/join", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	roomJoinHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response roomJoinResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Team.Name != "Taylor" {
		t.Fatalf("team name = %q, want %q", response.Team.Name, "Taylor")
	}
	if response.Team.Owner != "Taylor" {
		t.Fatalf("team owner = %q, want %q", response.Team.Owner, "Taylor")
	}
}

func TestRoomJoinClaimsUnassignedTeam(t *testing.T) {
	originalStore := dataStore
	originalRoom := draftRoom
	originalPubSub := ps
	defer func() {
		dataStore = originalStore
		draftRoom = originalRoom
		ps = originalPubSub
	}()

	store := dal.NewMemoryDAL()
	team, err := store.AddTeam("Draft Slot 1", "", "", "")
	if err != nil {
		t.Fatalf("AddTeam() failed: %v", err)
	}
	dataStore = store
	draftRoom = newRoomState("A123")
	ps = nil

	body := strings.NewReader(`{"code":"A123","username":"Taylor","teamId":"` + team.ID + `"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/room/join", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	roomJoinHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response roomJoinResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Team.Name != "Draft Slot 1" {
		t.Fatalf("team name = %q, want %q", response.Team.Name, "Draft Slot 1")
	}
	if response.Team.Owner != "Taylor" {
		t.Fatalf("team owner = %q, want %q", response.Team.Owner, "Taylor")
	}
}

func requestWithUser(request *http.Request, user *auth.User) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), "user", user)) //nolint:staticcheck
}
