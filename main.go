package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/handlers"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
)

var (
	templates *template.Template
	dataStore dal.DraftDAL
	ps        *pubsub.PubSub
)

func main() {
	// Initialize pubsub
	ps = pubsub.New()

	// Initialize data store based on DB_DRIVER env var
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		if os.Getenv("NODE_ENV") == "development" {
			dbDriver = "sqlite"
		} else {
			dbDriver = "memory"
		}
	}

	var err error
	switch dbDriver {
	case "sqlite":
		sqliteFile := os.Getenv("SQLITE_FILE")
		if sqliteFile == "" {
			sqliteFile = "dev.sqlite"
		}
		dataStore, err = dal.NewSQLiteDAL(sqliteFile)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
	case "memory":
		dataStore = dal.NewMemoryDAL()
	default:
		log.Printf("Unknown DB_DRIVER '%s', falling back to memory", dbDriver)
		dataStore = dal.NewMemoryDAL()
	}

	log.Printf("Using DB driver: %s", dbDriver)

	// Load templates
	templates = template.Must(template.ParseGlob("templates/*.html"))

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Page routes
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/start", startHandler)
	mux.HandleFunc("/draft", draftHandler)
	mux.HandleFunc("/admin", adminHandler)

	// API routes
	api := handlers.NewAPIHandlers(dataStore, ps)
	
	// Draft API
	mux.HandleFunc("/api/draft/state", api.GetDraftState)
	mux.HandleFunc("/api/draft/pick", api.DraftPick)
	mux.HandleFunc("/api/draft/reset", api.ResetDraft)
	
	// Teams API
	mux.HandleFunc("/api/teams", api.ListTeams)
	mux.HandleFunc("/api/teams/add", api.AddTeam)
	mux.HandleFunc("/api/teams/reorder", api.ReorderTeams)
	
	// Players API
	mux.HandleFunc("/api/players/add", api.AddPlayer)
	mux.HandleFunc("/api/players/points", api.SetPlayerPoints)
	mux.HandleFunc("/api/players/profile", api.GetPlayerProfile)
	
	// Chat API
	mux.HandleFunc("/api/chat/list", api.ListChat)
	mux.HandleFunc("/api/chat/send", api.SendChatMessage)
	mux.HandleFunc("/api/chat/react", api.AddReaction)
	
	// SSE for realtime updates
	mux.HandleFunc("/api/events", api.EventsSSE)

	// Health check
	mux.HandleFunc("/api/health", healthHandler)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	addr := "0.0.0.0:" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/start", http.StatusSeeOther)
}

func startHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Teams": state.Teams,
	}

	if err := templates.ExecuteTemplate(w, "start.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func draftHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Players": state.Players,
		"Teams":   state.Teams,
		"Chat":    state.Chat,
	}

	if err := templates.ExecuteTemplate(w, "draft.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Players": state.Players,
		"Teams":   state.Teams,
	}

	if err := templates.ExecuteTemplate(w, "admin.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
