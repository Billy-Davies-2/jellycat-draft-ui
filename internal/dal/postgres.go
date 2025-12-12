package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

// PostgresDAL implements DraftDAL using PostgreSQL
type PostgresDAL struct {
	db            *sql.DB
	reactionUsers map[string]map[string]map[string]bool
}

// NewPostgresDAL creates a new PostgreSQL data access layer optimized for CloudNativePG
func NewPostgresDAL(connString string) (*PostgresDAL, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	// CloudNativePG optimization: Configure connection pool settings
	// These settings are optimized for CloudNativePG high-availability clusters
	db.SetMaxOpenConns(25)                 // Limit max connections (CloudNativePG default max_connections is 100)
	db.SetMaxIdleConns(5)                  // Keep some idle connections for quick reuse
	db.SetConnMaxLifetime(5 * time.Minute) // Recycle connections to handle failovers gracefully
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections to reduce load

	// Test connection with retry logic for Kubernetes DNS resolution
	// Increased timeout to 60s to handle DNS propagation delays in Kubernetes
	maxRetries := 5
	retryDelay := 5 * time.Second
	var lastErr error
	
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		err := db.PingContext(ctx)
		cancel()
		
		if err == nil {
			// Connection successful
			break
		}
		
		lastErr = err
		if i < maxRetries-1 {
			// Wait before retrying (unless it's the last attempt)
			time.Sleep(retryDelay)
		}
	}
	
	if lastErr != nil {
		return nil, fmt.Errorf("failed to ping postgres after %d retries: %w", maxRetries, lastErr)
	}

	dal := &PostgresDAL{
		db:            db,
		reactionUsers: make(map[string]map[string]map[string]bool),
	}

	if err := dal.initSchema(); err != nil {
		return nil, err
	}

	return dal, nil
}

func (p *PostgresDAL) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS players (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		position TEXT NOT NULL,
		team TEXT NOT NULL,
		points INTEGER NOT NULL,
		cuddle_points INTEGER NOT NULL DEFAULT 50,
		tier TEXT NOT NULL,
		drafted BOOLEAN NOT NULL DEFAULT false,
		drafted_by TEXT,
		image TEXT,
		image_data BYTEA,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		owner TEXT NOT NULL,
		mascot TEXT NOT NULL,
		color TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS team_players (
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		player_id TEXT NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		player_data JSONB NOT NULL,
		draft_pick_number INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (team_id, player_id)
	);

	CREATE TABLE IF NOT EXISTS chat (
		id TEXT PRIMARY KEY,
		ts BIGINT NOT NULL,
		type TEXT NOT NULL,
		text TEXT NOT NULL,
		emotes JSONB NOT NULL DEFAULT '{}'::jsonb,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- CloudNativePG optimization: Add indexes for common query patterns
	CREATE INDEX IF NOT EXISTS idx_players_drafted ON players(drafted);
	CREATE INDEX IF NOT EXISTS idx_players_points ON players(points DESC);
	CREATE INDEX IF NOT EXISTS idx_chat_ts ON chat(ts);
	CREATE INDEX IF NOT EXISTS idx_team_players_team_id ON team_players(team_id);
	CREATE INDEX IF NOT EXISTS idx_teams_created_at ON teams(created_at);
	`

	if _, err := p.db.Exec(schema); err != nil {
		return err
	}

	// Add cuddle_points column to existing databases (migration)
	_, err := p.db.Exec(`
		ALTER TABLE players
		ADD COLUMN IF NOT EXISTS cuddle_points INTEGER NOT NULL DEFAULT 50
	`)
	if err != nil {
		return fmt.Errorf("failed to add cuddle_points column: %w", err)
	}

	// Add draft_pick_number column to team_players for existing databases
	_, err = p.db.Exec(`
		ALTER TABLE team_players
		ADD COLUMN IF NOT EXISTS draft_pick_number INTEGER
	`)
	if err != nil {
		return fmt.Errorf("failed to add draft_pick_number column: %w", err)
	}

	// Check if we need to seed data
	var count int
	if err := p.db.QueryRow("SELECT COUNT(*) FROM players").Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		if err := p.seedData(); err != nil {
			return err
		}
	}

	return nil
}

func (p *PostgresDAL) seedData() error {
	// CloudNativePG optimization: Use a transaction for batch inserts
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	players := getDefaultPlayers()
	teams := getDefaultTeams()

	// CloudNativePG optimization: Batch insert players to reduce round trips
	playerStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO players (id, name, position, team, points, tier, drafted, drafted_by, image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`)
	if err != nil {
		return err
	}
	defer playerStmt.Close()

	for _, player := range players {
		_, err := p.db.Exec(`
			INSERT INTO players (id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, player.ID, player.Name, player.Position, player.Team, player.Points, player.CuddlePoints, player.Tier, player.Drafted, "", player.Image)
		if err != nil {
			return err
		}
	}

	// CloudNativePG optimization: Batch insert teams
	teamStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO teams (id, name, owner, mascot, color)
		VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return err
	}
	defer teamStmt.Close()

	for _, team := range teams {
		_, err := teamStmt.ExecContext(ctx, team.ID, team.Name, team.Owner, team.Mascot, team.Color)
		if err != nil {
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return err
	}

	// Migrate images from static/images directory to database
	if err := p.MigrateImagesToDatabase(); err != nil {
		// Log warning but don't fail - images are optional
		fmt.Printf("Warning: Failed to migrate images to database: %v\n", err)
	}

	// Add welcome messages
	p.AddChatMessage("Welcome to the Jellycat Draft! 🎉", "system")
	p.AddChatMessage("Tip: Click a Jellycat card to draft it!", "system")
	p.AddChatMessage("Who will snag Bashful Bunny first? 🐰", "system")

	return nil
}

func (p *PostgresDAL) GetState() (*models.DraftState, error) {
	state := &models.DraftState{
		Players: []models.Player{},
		Teams:   []models.Team{},
		Chat:    []models.ChatMessage{},
	}

	// Get players
	rows, err := p.db.Query(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, COALESCE(drafted_by, ''), image
		FROM players
		ORDER BY points DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var player models.Player
		err := rows.Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.CuddlePoints, &player.Tier, &player.Drafted, &player.DraftedBy, &player.Image)
		if err != nil {
			return nil, err
		}
		state.Players = append(state.Players, player)
	}

	// CloudNativePG optimization: Get teams with their players in a single query using JOIN
	// This eliminates N+1 query problem and improves performance with read replicas
	teamRows, err := p.db.Query(`
		SELECT
			t.id, t.name, t.owner, t.mascot, t.color,
			tp.player_data
		FROM teams t
		LEFT JOIN team_players tp ON t.id = tp.team_id
		ORDER BY t.created_at, tp.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer teamRows.Close()

	// Build teams map to aggregate players
	teamsMap := make(map[string]*models.Team)
	teamOrder := []string{} // Track order of teams

	for teamRows.Next() {
		var teamID, teamName, teamOwner, teamMascot, teamColor string
		var playerJSON sql.NullString

		err := teamRows.Scan(&teamID, &teamName, &teamOwner, &teamMascot, &teamColor, &playerJSON)
		if err != nil {
			return nil, err
		}

		// Create team if not exists
		if _, exists := teamsMap[teamID]; !exists {
			teamsMap[teamID] = &models.Team{
				ID:      teamID,
				Name:    teamName,
				Owner:   teamOwner,
				Mascot:  teamMascot,
				Color:   teamColor,
				Players: []models.Player{},
			}
			teamOrder = append(teamOrder, teamID)
		}

		// Add player if player_data is not null (from LEFT JOIN)
		if playerJSON.Valid && playerJSON.String != "" {
			var player models.Player
			if err := json.Unmarshal([]byte(playerJSON.String), &player); err != nil {
				return nil, err
			}
			teamsMap[teamID].Players = append(teamsMap[teamID].Players, player)
		}
	}

	// Convert map to ordered slice
	for _, teamID := range teamOrder {
		state.Teams = append(state.Teams, *teamsMap[teamID])
	}

	// Get chat
	chatRows, err := p.db.Query(`SELECT id, ts, type, text, emotes FROM chat ORDER BY ts ASC`)
	if err != nil {
		return nil, err
	}
	defer chatRows.Close()

	for chatRows.Next() {
		var msg models.ChatMessage
		var emotesJSON []byte
		err := chatRows.Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
		if err != nil {
			return nil, err
		}
		msg.Emotes = make(map[string]int)
		json.Unmarshal(emotesJSON, &msg.Emotes)
		state.Chat = append(state.Chat, msg)
	}

	return state, nil
}

func (p *PostgresDAL) Reset() error {
	// Clear all tables
	_, err := p.db.Exec("TRUNCATE team_players, chat, teams, players CASCADE")
	if err != nil {
		return err
	}

	p.reactionUsers = make(map[string]map[string]map[string]bool)

	// Re-seed
	return p.seedData()
}

func (p *PostgresDAL) AddPlayer(player *models.Player) (*models.Player, error) {
	if player.ID == "" {
		player.ID = genID("player")
	}

	// Assign random cuddle points if not already set
	if player.CuddlePoints == 0 {
		player.CuddlePoints = randomCuddlePoints()
	}

	_, err := p.db.Exec(`
		INSERT INTO players (id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, player.ID, player.Name, player.Position, player.Team, player.Points, player.CuddlePoints, player.Tier, player.Drafted, player.DraftedBy, player.Image)

	return player, err
}

func (p *PostgresDAL) SetPlayerPoints(id string, points int) (*models.Player, error) {
	_, err := p.db.Exec(`UPDATE players SET points = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, points, id)
	if err != nil {
		return nil, err
	}

	// Also update in team_players
	_, err = p.db.Exec(`
		UPDATE team_players
		SET player_data = jsonb_set(player_data, '{points}', $1::text::jsonb)
		WHERE (player_data->>'id') = $2
	`, points, id)

	// Get updated player
	var player models.Player
	err = p.db.QueryRow(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, COALESCE(drafted_by, ''), image
		FROM players WHERE id = $1
	`, id).Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.CuddlePoints, &player.Tier, &player.Drafted, &player.DraftedBy, &player.Image)

	return &player, err
}

func (p *PostgresDAL) ReorderTeams(order []string) ([]models.Team, error) {
	// For Postgres, we can use a CTE with row numbers
	teams := []models.Team{}

	for _, id := range order {
		var t models.Team
		err := p.db.QueryRow(`
			SELECT id, name, owner, mascot, color
			FROM teams WHERE id = $1
		`, id).Scan(&t.ID, &t.Name, &t.Owner, &t.Mascot, &t.Color)

		if err == nil {
			t.Players = []models.Player{}
			teams = append(teams, t)
		}
	}

	return teams, nil
}

func (p *PostgresDAL) DraftPlayer(playerID, teamID string) error {
	// CloudNativePG optimization: Use context with timeout for better failover handling
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the current draft pick number (count of already drafted players + 1)
	var draftPickNumber int
	err = tx.QueryRow(`SELECT COUNT(*) + 1 FROM team_players`).Scan(&draftPickNumber)
	if err != nil {
		return err
	}

	// Get player including cuddle_points
	var player models.Player
	err = tx.QueryRow(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, image
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.CuddlePoints, &player.Tier, &player.Drafted, &player.Image)
	if err != nil {
		return err
	}

	if player.Drafted {
		return fmt.Errorf("player already drafted")
	}

	// Get team
	var teamName, teamMascot string
	err = tx.QueryRow(`SELECT name, mascot FROM teams WHERE id = $1`, teamID).Scan(&teamName, &teamMascot)
	if err != nil {
		return err
	}

	// Calculate cuddle points adjustment based on draft position
	// Early picks (1-6) gain points, late picks (13-18) lose points
	cuddlePointsAdjustment := 0
	if draftPickNumber <= 6 {
		// Early picks gain 8-18 points (pick 1 gets +18, pick 6 gets +8)
		cuddlePointsAdjustment = 20 - (draftPickNumber * 2)
	} else if draftPickNumber >= 13 {
		// Late picks lose 5-10 points (pick 13 loses -5, pick 18 loses -10)
		cuddlePointsAdjustment = 8 - draftPickNumber
	}

	newCuddlePoints := player.CuddlePoints + cuddlePointsAdjustment
	// Ensure cuddle points stay within reasonable bounds (min 10, max 100)
	if newCuddlePoints < 10 {
		newCuddlePoints = 10
	}
	if newCuddlePoints > 100 {
		newCuddlePoints = 100
	}

	// Update player as drafted with adjusted cuddle points
	_, err = tx.Exec(`
		UPDATE players
		SET drafted = true, drafted_by = $1, cuddle_points = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`, teamName, newCuddlePoints, playerID)
	if err != nil {
		return err
	}

	// Update player object for JSON storage
	player.Drafted = true
	player.DraftedBy = teamName
	player.CuddlePoints = newCuddlePoints
	playerJSON, err := json.Marshal(player)
	if err != nil {
		return fmt.Errorf("failed to marshal player data: %w", err)
	}

	// Add player to team with draft pick number
	_, err = tx.Exec(`
		INSERT INTO team_players (team_id, player_id, player_data, draft_pick_number)
		VALUES ($1, $2, $3, $4)
	`, teamID, playerID, playerJSON, draftPickNumber)
	if err != nil {
		return err
	}

	// Add chat message
	msg := fmt.Sprintf("%s %s drafted %s (%s • %s)", teamMascot, teamName, player.Name, player.Team, player.Position)
	emotesJSON, _ := json.Marshal(map[string]int{})
	_, err = tx.ExecContext(ctx, `
		INSERT INTO chat (id, ts, type, text, emotes)
		VALUES ($1, $2, $3, $4, $5)
	`, genID("msg"), time.Now().UnixMilli(), "system", msg, emotesJSON)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (p *PostgresDAL) AddChatMessage(text, msgType string) (*models.ChatMessage, error) {
	msg := &models.ChatMessage{
		ID:     genID("msg"),
		TS:     time.Now().UnixMilli(),
		Type:   msgType,
		Text:   text,
		Emotes: make(map[string]int),
	}

	emotesJSON, _ := json.Marshal(msg.Emotes)
	_, err := p.db.Exec(`
		INSERT INTO chat (id, ts, type, text, emotes)
		VALUES ($1, $2, $3, $4, $5)
	`, msg.ID, msg.TS, msg.Type, msg.Text, emotesJSON)

	return msg, err
}

func (p *PostgresDAL) AddReaction(messageID, emote, userID string) (*models.ChatMessage, error) {
	uid := userID
	if uid == "" {
		uid = "anon"
	}

	// Check if user already reacted
	if p.reactionUsers[messageID] == nil {
		p.reactionUsers[messageID] = make(map[string]map[string]bool)
	}
	if p.reactionUsers[messageID][emote] == nil {
		p.reactionUsers[messageID][emote] = make(map[string]bool)
	}

	if p.reactionUsers[messageID][emote][uid] {
		// Already reacted, return current message
		var msg models.ChatMessage
		var emotesJSON []byte
		err := p.db.QueryRow(`SELECT id, ts, type, text, emotes FROM chat WHERE id = $1`, messageID).Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(emotesJSON, &msg.Emotes)
		return &msg, nil
	}

	p.reactionUsers[messageID][emote][uid] = true

	// Update emotes using jsonb operations
	_, err := p.db.Exec(`
		UPDATE chat
		SET emotes = jsonb_set(
			COALESCE(emotes, '{}'::jsonb),
			ARRAY[$2],
			(COALESCE((emotes->>$2)::int, 0) + 1)::text::jsonb
		)
		WHERE id = $1
	`, messageID, emote)
	if err != nil {
		return nil, err
	}

	// Return updated message
	var msg models.ChatMessage
	var emotesJSON []byte
	err = p.db.QueryRow(`SELECT id, ts, type, text, emotes FROM chat WHERE id = $1`, messageID).Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(emotesJSON, &msg.Emotes)

	return &msg, nil
}

func (p *PostgresDAL) AddTeam(name, owner, mascot, color string) (*models.Team, error) {
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

	// Count existing teams for default mascot/color
	var count int
	p.db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count)

	if mascot == "" {
		mascot = mascots[count%len(mascots)]
	}
	if color == "" {
		color = colors[count%len(colors)]
	}

	team := &models.Team{
		ID:      genID("team"),
		Name:    name,
		Owner:   owner,
		Mascot:  mascot,
		Color:   color,
		Players: []models.Player{},
	}

	_, err := p.db.Exec(`
		INSERT INTO teams (id, name, owner, mascot, color)
		VALUES ($1, $2, $3, $4, $5)
	`, team.ID, team.Name, team.Owner, team.Mascot, team.Color)
	if err != nil {
		return nil, err
	}

	// Add system message
	msg := fmt.Sprintf("New team joined the draft: %s %s (Owner: %s)", team.Mascot, team.Name, team.Owner)
	p.AddChatMessage(msg, "system")

	return team, nil
}

func (p *PostgresDAL) Close() error {
	// CloudNativePG optimization: Gracefully close all connections
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}
