package dal

import "github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"

// DraftDAL defines the interface for data access layer
type DraftDAL interface {
	GetState() (*models.DraftState, error)
	Reset() error
	AddPlayer(player *models.Player) (*models.Player, error)
	UpdatePlayer(player *models.Player) (*models.Player, error)
	DeletePlayer(id string) error
	SetPlayerPoints(id string, points int) (*models.Player, error)
	ReorderTeams(order []string) ([]models.Team, error)
	DraftPlayer(playerID, teamID string) error
	AddChatMessage(text, msgType string) (*models.ChatMessage, error)
	AddReaction(messageID, emote, userID string) (*models.ChatMessage, error)
	AddTeam(name, owner, mascot, color string) (*models.Team, error)
}
