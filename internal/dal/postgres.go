package dal

import (
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

// NewPostgresDAL creates a new PostgreSQL data access layer
func NewPostgresDAL(connString string) (*PostgresDAL, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
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

	CREATE INDEX IF NOT EXISTS idx_players_drafted ON players(drafted);
	CREATE INDEX IF NOT EXISTS idx_chat_ts ON chat(ts);
	`

	if _, err := p.db.Exec(schema); err != nil {
		return err
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
	players := getDefaultPlayers()
	teams := getDefaultTeams()

	// Insert players
	for _, player := range players {
		_, err := p.db.Exec(`
			INSERT INTO players (id, name, position, team, points, tier, drafted, drafted_by, image)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, player.ID, player.Name, player.Position, player.Team, player.Points, player.Tier, player.Drafted, "", player.Image)
		if err != nil {
			return err
		}
	}

	// Insert teams
	for _, team := range teams {
		_, err := p.db.Exec(`
			INSERT INTO teams (id, name, owner, mascot, color)
			VALUES ($1, $2, $3, $4, $5)
		`, team.ID, team.Name, team.Owner, team.Mascot, team.Color)
		if err != nil {
			return err
		}
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
		SELECT id, name, position, team, points, tier, drafted, COALESCE(drafted_by, ''), image
		FROM players
		ORDER BY points DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var player models.Player
		err := rows.Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.Tier, &player.Drafted, &player.DraftedBy, &player.Image)
		if err != nil {
			return nil, err
		}
		state.Players = append(state.Players, player)
	}

	// Get teams with their players
	teamRows, err := p.db.Query(`SELECT id, name, owner, mascot, color FROM teams ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer teamRows.Close()

	for teamRows.Next() {
		var t models.Team
		err := teamRows.Scan(&t.ID, &t.Name, &t.Owner, &t.Mascot, &t.Color)
		if err != nil {
			return nil, err
		}
		t.Players = []models.Player{}

		// Get team players
		playerRows, err := p.db.Query(`SELECT player_data FROM team_players WHERE team_id = $1`, t.ID)
		if err != nil {
			return nil, err
		}

		for playerRows.Next() {
			var playerJSON []byte
			if err := playerRows.Scan(&playerJSON); err != nil {
				playerRows.Close()
				return nil, err
			}
			var p models.Player
			if err := json.Unmarshal(playerJSON, &p); err != nil {
				playerRows.Close()
				return nil, err
			}
			t.Players = append(t.Players, p)
		}
		playerRows.Close()

		state.Teams = append(state.Teams, t)
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

	_, err := p.db.Exec(`
		INSERT INTO players (id, name, position, team, points, tier, drafted, drafted_by, image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, player.ID, player.Name, player.Position, player.Team, player.Points, player.Tier, player.Drafted, player.DraftedBy, player.Image)

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
		SELECT id, name, position, team, points, tier, drafted, COALESCE(drafted_by, ''), image
		FROM players WHERE id = $1
	`, id).Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.Tier, &player.Drafted, &player.DraftedBy, &player.Image)

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
	tx, err := p.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get player
	var player models.Player
	err = tx.QueryRow(`
		SELECT id, name, position, team, points, tier, drafted, image
		FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&player.ID, &player.Name, &player.Position, &player.Team, &player.Points, &player.Tier, &player.Drafted, &player.Image)

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

	// Update player as drafted
	_, err = tx.Exec(`UPDATE players SET drafted = true, drafted_by = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, teamName, playerID)
	if err != nil {
		return err
	}

	// Add player to team
	player.Drafted = true
	player.DraftedBy = teamName
	playerJSON, _ := json.Marshal(player)
	_, err = tx.Exec(`
		INSERT INTO team_players (team_id, player_id, player_data)
		VALUES ($1, $2, $3)
	`, teamID, playerID, playerJSON)
	if err != nil {
		return err
	}

	// Add chat message
	msg := fmt.Sprintf("%s %s drafted %s (%s • %s)", teamMascot, teamName, player.Name, player.Team, player.Position)
	emotesJSON, _ := json.Marshal(map[string]int{})
	_, err = tx.Exec(`
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
	return p.db.Close()
}
