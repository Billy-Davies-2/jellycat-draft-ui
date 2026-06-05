package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/auth"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/clickhouse"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	grpcserver "github.com/Billy-Davies-2/jellycat-draft-ui/internal/grpc"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/handlers"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/pubsub"
	pb "github.com/Billy-Davies-2/jellycat-draft-ui/proto"
	"google.golang.org/grpc"
)

var (
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
	// Initialize logger first
	logger.Init()

	logger.Info("Starting Jellycat Draft microservice")
	environment := os.Getenv("ENVIRONMENT")

	// Initialize database driver
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		if environment == "production" {
			dbDriver = "postgres"
		} else {
			dbDriver = "memory"
		}
	}

	var err error
	switch dbDriver {
	case "memory":
		dataStore = dal.NewMemoryDAL()
		logger.Info("Using in-memory data store")
	case "sqlite":
		sqliteFile := os.Getenv("SQLITE_FILE")
		if sqliteFile == "" {
			sqliteFile = "dev.sqlite"
		}
		dataStore, err = dal.NewSQLiteDAL(sqliteFile)
		if err != nil {
			logger.Error("Failed to initialize SQLite", "error", err)
			log.Fatalf("Failed to initialize SQLite: %v", err)
		}
		logger.Info("Connected to SQLite database", "file", sqliteFile)
	case "postgres":
		dbConnString := os.Getenv("DATABASE_URL")
		if dbConnString == "" {
			dbConnString = buildPostgresURLFromEnv()
		}
		if dbConnString == "" {
			logger.Error("DATABASE_URL environment variable is required for postgres driver")
			log.Fatal("DATABASE_URL environment variable is required for postgres driver")
		}
		dataStore, err = dal.NewPostgresDAL(dbConnString)
		if err != nil {
			logger.Error("Failed to initialize Postgres", "error", err)
			log.Fatalf("Failed to initialize Postgres: %v", err)
		}
		logger.Info("Connected to Postgres database")
	default:
		logger.Error("Unknown DB_DRIVER", "driver", dbDriver)
		log.Fatalf("Unknown DB_DRIVER: %s (valid: memory, sqlite, postgres)", dbDriver)
	}

	// Initialize pub/sub (NATS JetStream or Embedded NATS for local development)
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	natsSubject := os.Getenv("NATS_SUBJECT")
	if natsSubject == "" {
		natsSubject = "draft.events"
	}

	var natsPubSub interface {
		Publish(pubsub.Event)
		Subscribe() chan pubsub.Event
		Unsubscribe(chan pubsub.Event)
	}

	// Use embedded NATS in development mode, real NATS in production
	if environment == "" || environment == "development" {
		logger.Info("Starting embedded NATS server for local development")
		embeddedNats, err := pubsub.NewEmbeddedNATSPubSub(pubsub.EmbeddedNATSOptions{
			Port:       0, // Random available port
			Subject:    natsSubject,
			StreamName: "DRAFT_EVENTS",
			StoreDir:   "", // In-memory storage
		})
		if err != nil {
			logger.Error("Failed to initialize embedded NATS", "error", err)
			log.Fatalf("Failed to initialize embedded NATS: %v", err)
		}
		natsPubSub = embeddedNats
		logger.Info("Embedded NATS server ready", "url", embeddedNats.GetServerURL())
	} else {
		logger.Info("Using real NATS JetStream for production")
		realNats, err := pubsub.NewNATSPubSub(natsURL, natsSubject)
		if err != nil {
			logger.Error("Failed to initialize NATS", "error", err)
			log.Fatalf("Failed to initialize NATS: %v", err)
		}
		natsPubSub = realNats
		logger.Info("Connected to NATS", "url", natsURL)
	}

	ps = natsPubSub
	draftRoom = newRoomState(os.Getenv("ROOM_CODE"))
	logger.Info("Draft room code ready", "code", draftRoom.Code())

	// Initialize ClickHouse client only when analytics is explicitly enabled.
	clickHouseEnabled := os.Getenv("CLICKHOUSE_ENABLED") == "true"
	var chErr error
	if clickHouseEnabled {
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

		chClient, chErr = clickhouse.NewClient(chAddr, chDB, chUser, chPass)
		if chErr != nil {
			logger.Error("Failed to initialize ClickHouse", "error", chErr, "address", chAddr)
			log.Fatalf("Failed to initialize ClickHouse: %v", chErr)
		}
		logger.Info("Connected to ClickHouse", "address", chAddr, "database", chDB)
	} else {
		logger.Info("Skipping ClickHouse analytics integration", "enabled", clickHouseEnabled)
	}

	// Start periodic cuddle points sync (only in production with ClickHouse)
	if chClient != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			// Initial sync
			syncCuddlePoints()

			for range ticker.C {
				syncCuddlePoints()
			}
		}()
	} else {
		logger.Info("Skipping cuddle points sync (ClickHouse not configured)")
	}

	// Initialize authentication
	// Use mock auth in development mode, Authentik OAuth2 in production
	if environment == "" || environment == "development" {
		logger.Info("Using mock authentication for local development (no Authentik server required)")
		authProvider = auth.NewMockAuth()
	} else {
		authentikBaseURL := os.Getenv("AUTHENTIK_BASE_URL")
		authentikClientID := os.Getenv("AUTHENTIK_CLIENT_ID")
		authentikClientSecret := os.Getenv("AUTHENTIK_CLIENT_SECRET")
		authentikRedirectURL := os.Getenv("AUTHENTIK_REDIRECT_URL")

		if authentikBaseURL == "" || authentikClientID == "" || authentikClientSecret == "" {
			logger.Error("AUTHENTIK_BASE_URL, AUTHENTIK_CLIENT_ID, and AUTHENTIK_CLIENT_SECRET environment variables are required for production")
			log.Fatal("AUTHENTIK_BASE_URL, AUTHENTIK_CLIENT_ID, and AUTHENTIK_CLIENT_SECRET environment variables are required for production")
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
		logger.Info("Connected to Authentik", "url", authentikBaseURL)
	}

	// Load templates
	if _, err := template.ParseGlob("templates/*.html"); err != nil {
		logger.Error("Failed to parse templates", "error", err)
		log.Fatalf("Failed to parse templates: %v", err)
	}
	logger.Info("Templates loaded successfully")

	// Start gRPC server in a goroutine
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	go func() {
		lis, err := net.Listen("tcp", "0.0.0.0:"+grpcPort)
		if err != nil {
			logger.Error("Failed to listen for gRPC", "error", err, "port", grpcPort)
			log.Fatalf("Failed to listen for gRPC: %v", err)
		}

		grpcServer := grpc.NewServer()
		pb.RegisterDraftServiceServer(grpcServer, grpcserver.NewServer(dataStore, convertPubSub(ps)))

		logger.Info("gRPC server starting", "address", "0.0.0.0:"+grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Failed to serve gRPC", "error", err)
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Image serving from database (fallback to static files if not in DB)
	mux.HandleFunc("/images/", serveImageHandler)

	// Auth routes (public)
	mux.HandleFunc("/auth/login", authProvider.LoginHandler)
	mux.HandleFunc("/auth/callback", authProvider.CallbackHandler)
	mux.HandleFunc("/auth/logout", authProvider.LogoutHandler)

	// Page routes
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/start", authProvider.OptionalMiddleware(startHandler))
	mux.HandleFunc("/draft", authProvider.OptionalMiddleware(draftHandler))
	mux.HandleFunc("/join", pickHandler)
	mux.HandleFunc("/pick", pickHandler)
	mux.HandleFunc("/results", authProvider.OptionalMiddleware(resultsHandler))
	mux.HandleFunc("/admin", authProvider.Middleware(adminHandler))

	// API routes
	api := handlers.NewAPIHandlers(dataStore, convertPubSub(ps))

	// Draft API
	mux.HandleFunc("/api/draft/state", api.GetDraftState)
	mux.HandleFunc("/api/draft/pick", requireRoomCode(api.DraftPick))
	mux.HandleFunc("/api/draft/reset", authProvider.Middleware(requireAdmin(api.ResetDraft)))
	mux.HandleFunc("/api/draft/settings", authProvider.Middleware(requireAdmin(api.UpdateDraftSettings)))
	mux.HandleFunc("/api/room", roomInfoHandler)
	mux.HandleFunc("/api/room/qr", roomQRHandler)
	mux.HandleFunc("/api/room/join", roomJoinHandler)

	// Teams API
	mux.HandleFunc("/api/teams", api.ListTeams)
	mux.HandleFunc("/api/teams/add", authProvider.Middleware(api.AddTeam))
	mux.HandleFunc("/api/teams/update", api.UpdateTeam)
	mux.HandleFunc("/api/teams/delete", api.DeleteTeam)
	mux.HandleFunc("/api/teams/reorder", api.ReorderTeams)

	// Players API
	mux.HandleFunc("/api/players/add", api.AddPlayer)
	mux.HandleFunc("/api/players/update", api.UpdatePlayer)
	mux.HandleFunc("/api/players/delete", api.DeletePlayer)
	mux.HandleFunc("/api/players/points", api.SetPlayerPoints)
	mux.HandleFunc("/api/players/profile", api.GetPlayerProfile)

	// Image upload API
	mux.HandleFunc("/api/images/upload", api.UploadImage)
	mux.HandleFunc("/api/images/list", api.ListImages)

	// Chat API
	mux.HandleFunc("/api/chat/list", api.ListChat)
	mux.HandleFunc("/api/chat/send", api.SendChatMessage)
	mux.HandleFunc("/api/chat/react", api.AddReaction)

	// SSE for realtime updates
	mux.HandleFunc("/api/events", api.EventsSSE)

	// Health check endpoints
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/healthz", livenessHandler) // Kubernetes liveness probe
	mux.HandleFunc("/readyz", readinessHandler) // Kubernetes readiness probe

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	addr := "0.0.0.0:" + port
	logger.Info("Server starting", "address", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("Server failed", "error", err)
		log.Fatal(err)
	}
}

func requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !auth.IsAdmin(auth.GetUser(r)) {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func requireRoomCode(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		code := r.Header.Get("X-Jellycat-Room-Code")
		if code == "" && strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			var payload struct {
				Code     string `json:"code"`
				RoomCode string `json:"roomCode"`
			}
			if err := json.Unmarshal(body, &payload); err == nil {
				code = payload.Code
				if code == "" {
					code = payload.RoomCode
				}
			}
		}

		if !draftRoom.Matches(code) {
			http.Error(w, "Invalid room code", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func buildPostgresURLFromEnv() string {
	host := os.Getenv("POSTGRES_HOST")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	database := os.Getenv("POSTGRES_DB")
	if host == "" || user == "" || database == "" {
		return ""
	}

	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	sslMode := os.Getenv("POSTGRES_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	postgresURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   database,
	}
	query := postgresURL.Query()
	query.Set("sslmode", sslMode)
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
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
		"Teams":       state.Teams,
		"Settings":    state.Settings,
		"ModeOptions": models.DraftModeOptions(),
		"User":        user,
		"IsAdmin":     auth.IsAdmin(user),
	}
	for key, value := range roomTemplateData(r) {
		data[key] = value
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

	// Find if user owns the team with current pick
	var userTeamID string
	var isUserTurn bool
	if user != nil {
		for _, team := range state.Teams {
			if team.Owner == user.Username || team.Owner == user.Name {
				userTeamID = team.ID
				if team.ID == state.CurrentTeamID {
					isUserTurn = true
				}
				break
			}
		}
	}

	data := map[string]interface{}{
		"Players":             state.Players,
		"Teams":               state.Teams,
		"Chat":                state.Chat,
		"Settings":            state.Settings,
		"ModeOptions":         models.DraftModeOptions(),
		"IsBingoMode":         state.Settings.Mode == models.DraftModeBingo,
		"IsWheelMode":         state.Settings.Mode == models.DraftModeWheel,
		"DraftOrder":          state.DraftOrder,
		"BingoBoard":          state.BingoBoard,
		"CurrentBingoPrompt":  state.CurrentBingoPrompt,
		"WheelSlots":          state.WheelSlots,
		"SuggestedPick":       state.SuggestedPick,
		"AnalyticsConfigured": chClient != nil,
		"User":                user,
		"IsAdmin":             auth.IsAdmin(user),
		"CurrentPick":         state.CurrentPick,
		"CurrentRound":        state.CurrentRound,
		"PickInRound":         state.PickInRound,
		"CurrentTeamID":       state.CurrentTeamID,
		"CurrentTeamName":     state.CurrentTeamName,
		"UserTeamID":          userTeamID,
		"IsUserTurn":          isUserTurn,
	}
	for key, value := range roomTemplateData(r) {
		data[key] = value
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

func pickHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	user := auth.GetUser(r)

	var userTeamID string
	var isUserTurn bool
	if user != nil {
		for _, team := range state.Teams {
			if team.Owner == user.Username || team.Owner == user.Name {
				userTeamID = team.ID
				if team.ID == state.CurrentTeamID {
					isUserTurn = true
				}
				break
			}
		}
	}

	data := map[string]interface{}{
		"Players":             state.Players,
		"Teams":               state.Teams,
		"Settings":            state.Settings,
		"IsBingoMode":         state.Settings.Mode == models.DraftModeBingo,
		"IsWheelMode":         state.Settings.Mode == models.DraftModeWheel,
		"CurrentBingoPrompt":  state.CurrentBingoPrompt,
		"DraftOrder":          state.DraftOrder,
		"WheelSlots":          state.WheelSlots,
		"SuggestedPick":       state.SuggestedPick,
		"AnalyticsConfigured": chClient != nil,
		"User":                user,
		"IsAdmin":             auth.IsAdmin(user),
		"CurrentPick":         state.CurrentPick,
		"CurrentRound":        state.CurrentRound,
		"PickInRound":         state.PickInRound,
		"CurrentTeamID":       state.CurrentTeamID,
		"CurrentTeamName":     state.CurrentTeamName,
		"UserTeamID":          userTeamID,
		"IsUserTurn":          isUserTurn,
		"InitialRoomCode":     normalizeRoomCode(r.URL.Query().Get("code")),
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/pick.html")
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
		"Players":             state.Players,
		"Teams":               state.Teams,
		"Settings":            state.Settings,
		"ModeOptions":         models.DraftModeOptions(),
		"AnalyticsConfigured": chClient != nil,
		"User":                user,
		"IsAdmin":             true,
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

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	state, err := dataStore.GetState()
	if err != nil {
		http.Error(w, "Failed to load state", http.StatusInternalServerError)
		return
	}

	user := auth.GetUser(r)

	// Calculate total points for each team and sort by points
	type TeamWithPoints struct {
		models.Team
		TotalPoints int
	}

	teamsWithPoints := make([]TeamWithPoints, 0, len(state.Teams))
	for _, team := range state.Teams {
		totalPoints := 0
		for _, player := range team.Players {
			totalPoints += player.Points
		}
		teamsWithPoints = append(teamsWithPoints, TeamWithPoints{
			Team:        team,
			TotalPoints: totalPoints,
		})
	}

	// Sort by total points descending
	for i := 0; i < len(teamsWithPoints)-1; i++ {
		for j := i + 1; j < len(teamsWithPoints); j++ {
			if teamsWithPoints[j].TotalPoints > teamsWithPoints[i].TotalPoints {
				teamsWithPoints[i], teamsWithPoints[j] = teamsWithPoints[j], teamsWithPoints[i]
			}
		}
	}

	// Get winner (team with most points)
	var winner *TeamWithPoints
	if len(teamsWithPoints) > 0 && teamsWithPoints[0].TotalPoints > 0 {
		winner = &teamsWithPoints[0]
	}

	data := map[string]interface{}{
		"Teams":   teamsWithPoints,
		"Winner":  winner,
		"User":    user,
		"IsAdmin": auth.IsAdmin(user),
	}

	tmpl, err := template.ParseFiles("templates/base.html", "templates/results.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	status := "ok"
	httpStatus := http.StatusOK
	checks := make(map[string]interface{})

	// Check database connectivity
	if dataStore != nil {
		_, err := dataStore.GetState()
		if err != nil {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			checks["database"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			checks["database"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		checks["database"] = map[string]interface{}{
			"status": "not_configured",
		}
	}

	// Check ClickHouse connectivity (only in production)
	environment := os.Getenv("ENVIRONMENT")
	if environment == "production" && chClient != nil {
		_, err := chClient.GetAllCuddlePoints()
		if err != nil {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
			checks["clickhouse"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			checks["clickhouse"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else if environment == "production" {
		checks["clickhouse"] = map[string]interface{}{
			"status": "not_configured",
		}
	}

	// Check NATS connectivity (only in production) - We can verify by trying to publish a test event
	if environment == "production" && ps != nil {
		// Just verify ps is available - actual connection health is handled internally by NATS
		checks["nats"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	_ = ctx // Context ready for future use

	response := map[string]interface{}{
		"status":    status,
		"timestamp": time.Now().Unix(),
		"checks":    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// livenessHandler handles Kubernetes liveness probes
// Returns 200 if the application is running (doesn't check dependencies)
func livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now().Unix(),
	})
}

// readinessHandler handles Kubernetes readiness probes
// Returns 200 if the application is ready to serve traffic (checks critical dependencies)
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	_ = ctx // Context ready for future use

	// Check database connectivity - this is critical for readiness
	if dataStore != nil {
		_, err := dataStore.GetState()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "not_ready",
				"reason":    "database_unavailable",
				"timestamp": time.Now().Unix(),
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Unix(),
	})
}

// serveImageHandler serves images from the database or falls back to static files
func serveImageHandler(w http.ResponseWriter, r *http.Request) {
	// Try to get image from database if the DAL supports image storage.
	if imageStore, ok := dataStore.(dal.ImageStore); ok {
		imageData, contentType, err := imageStore.GetImageByPath(r.URL.Path)
		if err == nil && len(imageData) > 0 {
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
			w.Write(imageData)
			return
		}
	}

	// Fallback to serving from static files
	http.ServeFile(w, r, "static"+r.URL.Path)
}

// syncCuddlePoints syncs cuddle points from ClickHouse
func syncCuddlePoints() {
	logger.Info("Syncing cuddle points from ClickHouse")
	ctx := context.Background()
	_ = ctx // Context ready for future use

	err := chClient.SyncCuddlePoints(func(playerID string, points int) error {
		_, err := dataStore.SetPlayerPoints(playerID, points)
		return err
	})
	if err != nil {
		logger.Error("Failed to sync cuddle points", "error", err)
	} else {
		logger.Info("Cuddle points synced successfully")
	}
}

// convertPubSub wraps the NATS pubsub to provide a local *pubsub.PubSub for handlers/gRPC
// This creates a bidirectional bridge: publishes go to NATS, and NATS events come to local subscribers
func convertPubSub(ps interface {
	Publish(pubsub.Event)
	Subscribe() chan pubsub.Event
	Unsubscribe(chan pubsub.Event)
}) *pubsub.PubSub {
	// Create a wrapper that publishes to NATS and has local subscribers
	wrapper := pubsub.NewWithUpstream(ps)

	return wrapper
}
