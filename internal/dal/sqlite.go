package dal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

// SQLiteDAL implements DraftDAL using SQLite
type SQLiteDAL struct {
	db            *sql.DB
	reactionUsers map[string]map[string]map[string]bool
}

// NewSQLiteDAL creates a new SQLite data access layer
func NewSQLiteDAL(dbPath string) (*SQLiteDAL, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	dal := &SQLiteDAL{
		db:            db,
		reactionUsers: make(map[string]map[string]map[string]bool),
	}

	if err := dal.initSchema(); err != nil {
		return nil, err
	}

	return dal, nil
}

func (s *SQLiteDAL) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS players (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		position TEXT NOT NULL,
		team TEXT NOT NULL,
		points INTEGER NOT NULL,
		cuddle_points INTEGER NOT NULL DEFAULT 50,
		tier TEXT NOT NULL,
		drafted INTEGER NOT NULL DEFAULT 0,
		drafted_by TEXT,
		image TEXT
	);

	CREATE TABLE IF NOT EXISTS teams (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		owner TEXT NOT NULL,
		mascot TEXT NOT NULL,
		color TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS team_players (
		team_id TEXT NOT NULL,
		player_id TEXT NOT NULL,
		player_data TEXT NOT NULL,
		draft_pick_number INTEGER,
		FOREIGN KEY (team_id) REFERENCES teams(id),
		FOREIGN KEY (player_id) REFERENCES players(id)
	);

	CREATE TABLE IF NOT EXISTS chat (
		id TEXT PRIMARY KEY,
		ts INTEGER NOT NULL,
		type TEXT NOT NULL,
		text TEXT NOT NULL,
		emotes TEXT NOT NULL
	);
	`

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Add cuddle_points column to existing databases (migration)
	// SQLite doesn't support IF NOT EXISTS for ALTER TABLE, so we check first
	var cuddlePointsExists int
	err := s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('players') 
		WHERE name='cuddle_points'
	`).Scan(&cuddlePointsExists)
	if err != nil {
		return fmt.Errorf("failed to check cuddle_points column existence: %w", err)
	}

	if cuddlePointsExists == 0 {
		_, err = s.db.Exec(`ALTER TABLE players ADD COLUMN cuddle_points INTEGER NOT NULL DEFAULT 50`)
		if err != nil {
			return fmt.Errorf("failed to add cuddle_points column: %w", err)
		}
	}

	// Add draft_pick_number column to team_players for existing databases
	var draftPickNumberExists int
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM pragma_table_info('team_players') 
		WHERE name='draft_pick_number'
	`).Scan(&draftPickNumberExists)
	if err != nil {
		return fmt.Errorf("failed to check draft_pick_number column existence: %w", err)
	}

	if draftPickNumberExists == 0 {
		_, err = s.db.Exec(`ALTER TABLE team_players ADD COLUMN draft_pick_number INTEGER`)
		if err != nil {
			return fmt.Errorf("failed to add draft_pick_number column: %w", err)
		}
	}

	// Seed default data if empty
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM players").Scan(&count); err != nil {
		return err
	}

	if count == 0 {
		if err := s.seedData(); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteDAL) seedData() error {
	players := getDefaultPlayers()
	teams := getDefaultTeams()

	// Insert players
	for _, p := range players {
		_, err := s.db.Exec(`
			INSERT INTO players (id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, p.ID, p.Name, p.Position, p.Team, p.Points, p.CuddlePoints, p.Tier, 0, "", p.Image)
		if err != nil {
			return err
		}
	}

	// Insert teams
	for _, t := range teams {
		_, err := s.db.Exec(`
			INSERT INTO teams (id, name, owner, mascot, color)
			VALUES (?, ?, ?, ?, ?)
		`, t.ID, t.Name, t.Owner, t.Mascot, t.Color)
		if err != nil {
			return err
		}
	}

	// Add welcome messages
	s.AddChatMessage("Welcome to the Jellycat Draft! 🎉", "system")
	s.AddChatMessage("Tip: Click a Jellycat card to draft it!", "system")
	s.AddChatMessage("Who will snag Bashful Bunny first? 🐰", "system")

	return nil
}

func (s *SQLiteDAL) GetState() (*models.DraftState, error) {
	state := &models.DraftState{
		Players: []models.Player{},
		Teams:   []models.Team{},
		Chat:    []models.ChatMessage{},
	}

	// Get players
	rows, err := s.db.Query(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image
		FROM players
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p models.Player
		var drafted int
		var draftedBy sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &p.Position, &p.Team, &p.Points, &p.CuddlePoints, &p.Tier, &drafted, &draftedBy, &p.Image)
		if err != nil {
			return nil, err
		}
		p.Drafted = drafted == 1
		if draftedBy.Valid {
			p.DraftedBy = draftedBy.String
		}
		state.Players = append(state.Players, p)
	}

	// Get teams with their players
	teamRows, err := s.db.Query(`
		SELECT id, name, owner, mascot, color
		FROM teams
	`)
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
		playerRows, err := s.db.Query(`
			SELECT player_data FROM team_players WHERE team_id = ?
		`, t.ID)
		if err != nil {
			return nil, err
		}

		for playerRows.Next() {
			var playerJSON string
			if err := playerRows.Scan(&playerJSON); err != nil {
				playerRows.Close()
				return nil, err
			}
			var p models.Player
			if err := json.Unmarshal([]byte(playerJSON), &p); err != nil {
				playerRows.Close()
				return nil, err
			}
			t.Players = append(t.Players, p)
		}
		playerRows.Close()

		state.Teams = append(state.Teams, t)
	}

	// Get chat
	chatRows, err := s.db.Query(`
		SELECT id, ts, type, text, emotes
		FROM chat ORDER BY ts ASC
	`)
	if err != nil {
		return nil, err
	}
	defer chatRows.Close()

	for chatRows.Next() {
		var msg models.ChatMessage
		var emotesJSON string
		err := chatRows.Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
		if err != nil {
			return nil, err
		}
		msg.Emotes = make(map[string]int)
		json.Unmarshal([]byte(emotesJSON), &msg.Emotes)
		state.Chat = append(state.Chat, msg)
	}

	return state, nil
}

func (s *SQLiteDAL) Reset() error {
	// Clear all tables
	_, err := s.db.Exec("DELETE FROM team_players")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM chat")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM teams")
	if err != nil {
		return err
	}
	_, err = s.db.Exec("DELETE FROM players")
	if err != nil {
		return err
	}

	s.reactionUsers = make(map[string]map[string]map[string]bool)

	// Re-seed
	return s.seedData()
}

func (s *SQLiteDAL) AddPlayer(player *models.Player) (*models.Player, error) {
	if player.ID == "" {
		player.ID = genID("player")
	}

	// Assign random cuddle points if not already set
	if player.CuddlePoints == 0 {
		player.CuddlePoints = randomCuddlePoints()
	}

	drafted := 0
	if player.Drafted {
		drafted = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO players (id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, player.ID, player.Name, player.Position, player.Team, player.Points, player.CuddlePoints, player.Tier, drafted, player.DraftedBy, player.Image)

	return player, err
}

func (s *SQLiteDAL) SetPlayerPoints(id string, points int) (*models.Player, error) {
	_, err := s.db.Exec(`UPDATE players SET points = ? WHERE id = ?`, points, id)
	if err != nil {
		return nil, err
	}

	// Also update in team_players
	_, err = s.db.Exec(`
		UPDATE team_players 
		SET player_data = json_set(player_data, '$.points', ?) 
		WHERE json_extract(player_data, '$.id') = ?
	`, points, id)

	// Get updated player
	var p models.Player
	var drafted int
	var draftedBy sql.NullString
	err = s.db.QueryRow(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, drafted_by, image
		FROM players WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.Position, &p.Team, &p.Points, &p.CuddlePoints, &p.Tier, &drafted, &draftedBy, &p.Image)

	if err != nil {
		return nil, err
	}

	p.Drafted = drafted == 1
	if draftedBy.Valid {
		p.DraftedBy = draftedBy.String
	}

	return &p, nil
}

func (s *SQLiteDAL) ReorderTeams(order []string) ([]models.Team, error) {
	// SQLite doesn't have a direct way to reorder, so we'll use a temp table
	// For simplicity, we'll just return teams in the requested order
	teams := []models.Team{}

	for _, id := range order {
		var t models.Team
		err := s.db.QueryRow(`
			SELECT id, name, owner, mascot, color
			FROM teams WHERE id = ?
		`, id).Scan(&t.ID, &t.Name, &t.Owner, &t.Mascot, &t.Color)

		if err == nil {
			t.Players = []models.Player{}
			teams = append(teams, t)
		}
	}

	return teams, nil
}

func (s *SQLiteDAL) DraftPlayer(playerID, teamID string) error {
	tx, err := s.db.Begin()
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
	var p models.Player
	var drafted int
	err = tx.QueryRow(`
		SELECT id, name, position, team, points, cuddle_points, tier, drafted, image
		FROM players WHERE id = ?
	`, playerID).Scan(&p.ID, &p.Name, &p.Position, &p.Team, &p.Points, &p.CuddlePoints, &p.Tier, &drafted, &p.Image)

	if err != nil {
		return err
	}

	if drafted == 1 {
		return fmt.Errorf("player already drafted")
	}

	// Get team
	var teamName, teamMascot string
	err = tx.QueryRow(`
		SELECT name, mascot FROM teams WHERE id = ?
	`, teamID).Scan(&teamName, &teamMascot)

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

	newCuddlePoints := p.CuddlePoints + cuddlePointsAdjustment
	// Ensure cuddle points stay within reasonable bounds (min 10, max 100)
	if newCuddlePoints < 10 {
		newCuddlePoints = 10
	}
	if newCuddlePoints > 100 {
		newCuddlePoints = 100
	}

	// Update player as drafted with adjusted cuddle points
	_, err = tx.Exec(`
		UPDATE players SET drafted = 1, drafted_by = ?, cuddle_points = ? WHERE id = ?
	`, teamName, newCuddlePoints, playerID)
	if err != nil {
		return err
	}

	// Update player object for JSON storage
	p.Drafted = true
	p.DraftedBy = teamName
	p.CuddlePoints = newCuddlePoints
	playerJSON, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal player data: %w", err)
	}

	// Add player to team with draft pick number
	_, err = tx.Exec(`
		INSERT INTO team_players (team_id, player_id, player_data, draft_pick_number)
		VALUES (?, ?, ?, ?)
	`, teamID, playerID, string(playerJSON), draftPickNumber)
	if err != nil {
		return err
	}

	// Add chat message
	msg := fmt.Sprintf("%s %s drafted %s (%s • %s)", teamMascot, teamName, p.Name, p.Team, p.Position)
	emotesJSON, _ := json.Marshal(map[string]int{})
	_, err = tx.Exec(`
		INSERT INTO chat (id, ts, type, text, emotes)
		VALUES (?, ?, ?, ?, ?)
	`, genID("msg"), time.Now().UnixMilli(), "system", msg, string(emotesJSON))
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *SQLiteDAL) AddChatMessage(text, msgType string) (*models.ChatMessage, error) {
	msg := &models.ChatMessage{
		ID:     genID("msg"),
		TS:     time.Now().UnixMilli(),
		Type:   msgType,
		Text:   text,
		Emotes: make(map[string]int),
	}

	emotesJSON, _ := json.Marshal(msg.Emotes)
	_, err := s.db.Exec(`
		INSERT INTO chat (id, ts, type, text, emotes)
		VALUES (?, ?, ?, ?, ?)
	`, msg.ID, msg.TS, msg.Type, msg.Text, string(emotesJSON))

	return msg, err
}

func (s *SQLiteDAL) AddReaction(messageID, emote, userID string) (*models.ChatMessage, error) {
	uid := userID
	if uid == "" {
		uid = "anon"
	}

	// Check if user already reacted
	if s.reactionUsers[messageID] == nil {
		s.reactionUsers[messageID] = make(map[string]map[string]bool)
	}
	if s.reactionUsers[messageID][emote] == nil {
		s.reactionUsers[messageID][emote] = make(map[string]bool)
	}

	if s.reactionUsers[messageID][emote][uid] {
		// Already reacted, just return current message
		var msg models.ChatMessage
		var emotesJSON string
		err := s.db.QueryRow(`
			SELECT id, ts, type, text, emotes FROM chat WHERE id = ?
		`, messageID).Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(emotesJSON), &msg.Emotes)
		return &msg, nil
	}

	s.reactionUsers[messageID][emote][uid] = true

	// Get current emotes
	var emotesJSON string
	err := s.db.QueryRow(`SELECT emotes FROM chat WHERE id = ?`, messageID).Scan(&emotesJSON)
	if err != nil {
		return nil, err
	}

	emotes := make(map[string]int)
	json.Unmarshal([]byte(emotesJSON), &emotes)
	emotes[emote]++

	newEmotesJSON, _ := json.Marshal(emotes)
	_, err = s.db.Exec(`UPDATE chat SET emotes = ? WHERE id = ?`, string(newEmotesJSON), messageID)
	if err != nil {
		return nil, err
	}

	// Return updated message
	var msg models.ChatMessage
	err = s.db.QueryRow(`
		SELECT id, ts, type, text, emotes FROM chat WHERE id = ?
	`, messageID).Scan(&msg.ID, &msg.TS, &msg.Type, &msg.Text, &emotesJSON)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(emotesJSON), &msg.Emotes)

	return &msg, nil
}

func (s *SQLiteDAL) AddTeam(name, owner, mascot, color string) (*models.Team, error) {
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
	s.db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count)

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

	_, err := s.db.Exec(`
		INSERT INTO teams (id, name, owner, mascot, color)
		VALUES (?, ?, ?, ?, ?)
	`, team.ID, team.Name, team.Owner, team.Mascot, team.Color)

	if err != nil {
		return nil, err
	}

	// Add system message
	msg := fmt.Sprintf("New team joined the draft: %s %s (Owner: %s)", team.Mascot, team.Name, team.Owner)
	s.AddChatMessage(msg, "system")

	return team, nil
}

func (s *SQLiteDAL) Close() error {
	return s.db.Close()
}
