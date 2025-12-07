package main

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/auth"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/clickhouse"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	grpcserver "github.com/Billy-Davies-2/jellycat-draft-ui/internal/grpc"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/handlers"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/mocks"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
	pb "github.com/Billy-Davies-2/jellycat-draft-ui/proto"
	"google.golang.org/grpc"
)

var (
	templates    *template.Template
	dataStore    dal.DraftDAL
	authProvider auth.AuthProvider
	ps           interface {
		Publish(pubsub.Event)
		Subscribe() chan pubsub.Event
		Unsubscribe(chan pubsub.Event)
	}
	chClient interface {
		GetCuddlePoints(string) (int, error)
		GetAllCuddlePoints() (map[string]int, error)
		SyncCuddlePoints(func(string, int) error) error
		Close() error
	}
)

func main() {
	log.Println("Starting Jellycat Draft microservice")

	// Initialize database driver
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "memory"
	}

	var err error
	switch dbDriver {
	case "memory":
		dataStore = dal.NewMemoryDAL()
		log.Println("Using in-memory data store")
	case "sqlite":
		sqliteFile := os.Getenv("SQLITE_FILE")
		if sqliteFile == "" {
			sqliteFile = "dev.sqlite"
		}
		dataStore, err = dal.NewSQLiteDAL(sqliteFile)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
		log.Printf("Connected to SQLite database at %s", sqliteFile)
	case "postgres":
		dbConnString := os.Getenv("DATABASE_URL")
		if dbConnString == "" {
			log.Fatal("DATABASE_URL environment variable is required for postgres driver")
		}
		dataStore, err = dal.NewPostgresDAL(dbConnString)
		if err != nil {
			log.Fatalf("Failed to initialize Postgres: %v", err)
		}
		log.Println("Connected to Postgres database")
	default:
		log.Fatalf("Unknown DB_DRIVER: %s (valid: memory, sqlite, postgres)", dbDriver)
	}

	// Initialize pub/sub (NATS JetStream)
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	natsSubject := os.Getenv("NATS_SUBJECT")
	if natsSubject == "" {
		natsSubject = "draft.events"
	}

	natsPubSub, err := pubsub.NewNATSPubSub(natsURL, natsSubject)
	if err != nil {
		log.Fatalf("Failed to initialize NATS: %v", err)
	}
	ps = natsPubSub
	log.Printf("Connected to NATS at %s", natsURL)

	// Initialize ClickHouse client
	chAddr := os.Getenv("CLICKHOUSE_ADDR")
	if chAddr == "" {
		chAddr = "localhost:9000"
	}
	chDB := os.Getenv("CLICKHOUSE_DB")
	if chDB == "" {
		chDB = "default"
	}
	chUser := os.Getenv("CLICKHOUSE_USER")
	if chUser == "" {
		chUser = "default"
	}
	chPass := os.Getenv("CLICKHOUSE_PASSWORD")

	chClient, err = clickhouse.NewClient(chAddr, chDB, chUser, chPass)
	if err != nil {
		log.Fatalf("Failed to initialize ClickHouse: %v", err)
	}
	log.Printf("Connected to ClickHouse at %s", chAddr)

	// Start periodic cuddle points sync (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// Initial sync
		syncCuddlePoints()

		for range ticker.C {
			syncCuddlePoints()
		}
	}()

	// Initialize authentication (Authentik OAuth2)
	authentikBaseURL := os.Getenv("AUTHENTIK_BASE_URL")
	authentikClientID := os.Getenv("AUTHENTIK_CLIENT_ID")
	authentikClientSecret := os.Getenv("AUTHENTIK_CLIENT_SECRET")
	authentikRedirectURL := os.Getenv("AUTHENTIK_REDIRECT_URL")

	if authentikBaseURL == "" || authentikClientID == "" || authentikClientSecret == "" {
		log.Fatal("AUTHENTIK_BASE_URL, AUTHENTIK_CLIENT_ID, and AUTHENTIK_CLIENT_SECRET environment variables are required")
	}

	if authentikRedirectURL == "" {
		authentikRedirectURL = "http://localhost:3000/auth/callback"
	}

	authProvider = auth.NewAuthentikAuth(&auth.AuthentikConfig{
		BaseURL:      authentikBaseURL,
		ClientID:     authentikClientID,
		ClientSecret: authentikClientSecret,
		RedirectURL:  authentikRedirectURL,
		Scopes:       []string{"openid", "profile", "email"},
	})
	log.Printf("Connected to Authentik at %s", authentikBaseURL)

	// Load templates
	var tmplErr error
	templates, tmplErr = template.ParseGlob("templates/*.html")
	if tmplErr != nil {
		log.Fatalf("Failed to parse templates: %v", tmplErr)
	}
	log.Printf("Templates loaded successfully")

	// Start gRPC server in a goroutine
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	
	go func() {
		lis, err := net.Listen("tcp", "0.0.0.0:"+grpcPort)
		if err != nil {
			log.Fatalf("Failed to listen for gRPC: %v", err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterDraftServiceServer(grpcServer, grpcserver.NewServer(dataStore, convertPubSub(ps)))

		log.Printf("gRPC server starting on 0.0.0.0:%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Auth routes (public)
	mux.HandleFunc("/auth/login", authProvider.LoginHandler)
	mux.HandleFunc("/auth/callback", authProvider.CallbackHandler)
	mux.HandleFunc("/auth/logout", authProvider.LogoutHandler)

	// Page routes (protected)
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/start", authProvider.Middleware(startHandler))
	mux.HandleFunc("/draft", authProvider.Middleware(draftHandler))
	mux.HandleFunc("/admin", authProvider.Middleware(adminHandler))

	// API routes
	api := handlers.NewAPIHandlers(dataStore, convertPubSub(ps))
	
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

	user := auth.GetUser(r)
	data := map[string]interface{}{
		"Teams":   state.Teams,
		"User":    user,
		"IsAdmin": auth.IsAdmin(user),
	}

	// Parse both base and content templates
	tmpl, err := template.ParseFiles("templates/base.html", "templates/start.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func draftHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	user := auth.GetUser(r)
	data := map[string]interface{}{
		"Players": state.Players,
		"Teams":   state.Teams,
		"Chat":    state.Chat,
		"User":    user,
		"IsAdmin": auth.IsAdmin(user),
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/draft.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := auth.GetUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is an admin
	if !auth.IsAdmin(user) {
		http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
		return
	}

	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Players": state.Players,
		"Teams":   state.Teams,
		"User":    user,
		"IsAdmin": true,
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/admin.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
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

// syncCuddlePoints syncs cuddle points from ClickHouse
func syncCuddlePoints() {
	log.Println("Syncing cuddle points from ClickHouse...")
	ctx := context.Background()
	_ = ctx // Context ready for future use

	err := chClient.SyncCuddlePoints(func(playerID string, points int) error {
		_, err := dataStore.SetPlayerPoints(playerID, points)
		return err
	})
	if err != nil {
		log.Printf("Failed to sync cuddle points: %v", err)
	} else {
		log.Println("Cuddle points synced successfully")
	}
}

// convertPubSub converts the generic pubsub interface to *pubsub.PubSub for gRPC server
func convertPubSub(ps interface {
	Publish(pubsub.Event)
	Subscribe() chan pubsub.Event
	Unsubscribe(chan pubsub.Event)
}) *pubsub.PubSub {
	// If it's already a *pubsub.PubSub, return it
	if p, ok := ps.(*pubsub.PubSub); ok {
		return p
	}
	// If it's a mock, extract the embedded PubSub
	if m, ok := ps.(*mocks.MockNATSPubSub); ok {
		return m.PubSub
	}
	// Create a wrapper
	wrapper := pubsub.New()
	go func() {
		ch := ps.Subscribe()
		for event := range ch {
			wrapper.Publish(event)
		}
	}()
	return wrapper
}
