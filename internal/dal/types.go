package dal

import "github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"

// DraftDAL defines the interface for data access layer
type DraftDAL interface {
	GetState() (*models.DraftState, error)
	Reset() error
	SetDraftMode(mode models.DraftMode) (*models.DraftSettings, error)
	AddPlayer(player *models.Player) (*models.Player, error)
	UpdatePlayer(player *models.Player) (*models.Player, error)
	DeletePlayer(id string) error
	SetPlayerPoints(id string, points int) (*models.Player, error)
	ReorderTeams(order []string) ([]models.Team, error)
	DraftPlayer(playerID, teamID string) error
	AddChatMessage(text, msgType string) (*models.ChatMessage, error)
	AddReaction(messageID, emote, userID string) (*models.ChatMessage, error)
	AddTeam(name, owner, mascot, color string) (*models.Team, error)
	UpdateTeam(id, name, owner, mascot, color string) (*models.Team, error)
	DeleteTeam(id string) error
}

// ImageStore stores user-managed image assets outside the application image.
type ImageStore interface {
	GetImageByPath(path string) ([]byte, string, error)
	SaveImage(path, contentType string, data []byte) error
	ListImages() ([]string, error)
}
