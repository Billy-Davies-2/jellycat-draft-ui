package fuzz

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/handlers"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
)

func init() {
	// Initialize logger for tests
	logger.Init()
}

// FuzzHTTPDraftPick fuzzes the HTTP draft pick endpoint
func FuzzHTTPDraftPick(f *testing.F) {
	// Seed corpus with valid examples
	f.Add(`{"playerId":"1","teamId":"1"}`)
	f.Add(`{"playerId":"2","teamId":"2"}`)
	f.Add(`{"playerId":"invalid","teamId":"999"}`)

	f.Fuzz(func(t *testing.T, data string) {
		// Setup
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		// Create request
		req := httptest.NewRequest(http.MethodPost, "/api/draft/pick", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Execute
		api.DraftPick(w, req)

		// Should not panic - that's the main goal of fuzzing
		// We don't care if it returns an error, just that it doesn't crash
	})
}

// FuzzHTTPAddTeam fuzzes the HTTP add team endpoint
func FuzzHTTPAddTeam(f *testing.F) {
	// Seed corpus
	f.Add(`{"name":"Test Team","owner":"Owner"}`)
	f.Add(`{"name":"A","owner":""}`)
	f.Add(`{"name":"","owner":"X"}`)

	f.Fuzz(func(t *testing.T, data string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		req := httptest.NewRequest(http.MethodPost, "/api/teams/add", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.AddTeam(w, req)
	})
}

// FuzzHTTPSendChat fuzzes the HTTP send chat endpoint
func FuzzHTTPSendChat(f *testing.F) {
	// Seed corpus
	f.Add(`{"text":"Hello world","type":"user"}`)
	f.Add(`{"text":"","type":"system"}`)
	f.Add(`{"text":"` + string(make([]byte, 10000)) + `"}`)

	f.Fuzz(func(t *testing.T, data string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		req := httptest.NewRequest(http.MethodPost, "/api/chat/send", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.SendChatMessage(w, req)
	})
}

// FuzzHTTPSetPlayerPoints fuzzes the HTTP set player points endpoint
func FuzzHTTPSetPlayerPoints(f *testing.F) {
	// Seed corpus
	f.Add(`{"id":"1","points":100}`)
	f.Add(`{"id":"999","points":-1}`)
	f.Add(`{"id":"","points":0}`)

	f.Fuzz(func(t *testing.T, data string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		req := httptest.NewRequest(http.MethodPost, "/api/players/points", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.SetPlayerPoints(w, req)
	})
}

// FuzzHTTPReorderTeams fuzzes the HTTP reorder teams endpoint
func FuzzHTTPReorderTeams(f *testing.F) {
	// Seed corpus
	f.Add(`{"order":["1","2","3"]}`)
	f.Add(`{"order":[]}`)
	f.Add(`{"order":["invalid","999"]}`)

	f.Fuzz(func(t *testing.T, data string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		req := httptest.NewRequest(http.MethodPost, "/api/teams/reorder", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.ReorderTeams(w, req)
	})
}

// FuzzHTTPAddReaction fuzzes the HTTP add reaction endpoint
func FuzzHTTPAddReaction(f *testing.F) {
	// Seed corpus
	f.Add(`{"messageId":"msg_1","emote":"🎉","user":"user1"}`)
	f.Add(`{"messageId":"","emote":"","user":""}`)

	f.Fuzz(func(t *testing.T, data string) {
		dal := dal.NewMemoryDAL()
		ps := pubsub.New()
		api := handlers.NewAPIHandlers(dal, ps)

		// First add a message
		dal.AddChatMessage("Test message", "user")

		req := httptest.NewRequest(http.MethodPost, "/api/chat/react", bytes.NewBufferString(data))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		api.AddReaction(w, req)
	})
}

// FuzzJSONParsing fuzzes general JSON parsing
func FuzzJSONParsing(f *testing.F) {
	// Seed various JSON structures
	f.Add(`{"key":"value"}`)
	f.Add(`[1,2,3]`)
	f.Add(`null`)
	f.Add(`"string"`)
	f.Add(`123`)
	f.Add(`true`)

	f.Fuzz(func(t *testing.T, data string) {
		var result interface{}
		// Should not panic on any input
		json.Unmarshal([]byte(data), &result)
	})
}
