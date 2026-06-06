package dal

import (
	"path/filepath"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

func TestSQLiteDraftPlayerUsesPersistedTeamOrder(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")

	store, err := NewSQLiteDAL(filepath.Join(t.TempDir(), "draft.sqlite"))
	if err != nil {
		t.Fatalf("NewSQLiteDAL() failed: %v", err)
	}

	firstTeam, err := store.AddTeam("First", "First", "", "")
	if err != nil {
		t.Fatalf("AddTeam(first) failed: %v", err)
	}
	secondTeam, err := store.AddTeam("Second", "Second", "", "")
	if err != nil {
		t.Fatalf("AddTeam(second) failed: %v", err)
	}
	player, err := store.AddPlayer(&models.Player{
		Name:         "Turn Order Pick",
		Position:     "CC",
		Team:         "Test",
		Points:       100,
		CuddlePoints: 80,
		Tier:         models.TierA,
	})
	if err != nil {
		t.Fatalf("AddPlayer() failed: %v", err)
	}

	if _, err := store.ReorderTeams([]string{secondTeam.ID, firstTeam.ID}); err != nil {
		t.Fatalf("ReorderTeams() failed: %v", err)
	}

	state, err := store.GetState()
	if err != nil {
		t.Fatalf("GetState() failed: %v", err)
	}
	if state.CurrentTeamID != secondTeam.ID {
		t.Fatalf("CurrentTeamID = %q, want %q", state.CurrentTeamID, secondTeam.ID)
	}

	if err := store.DraftPlayer(player.ID, secondTeam.ID); err != nil {
		t.Fatalf("DraftPlayer() should allow reordered first team: %v", err)
	}
}
