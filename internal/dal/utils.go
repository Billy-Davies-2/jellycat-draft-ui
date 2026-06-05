package dal

import (
	"crypto/rand"
	"hash/fnv"
	"math/big"
	"os"
	"strings"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

// SeedDefaultCatalogEnabled returns true when the canned Jellycat catalog should be inserted into an empty store.
func SeedDefaultCatalogEnabled() bool {
	return IsDevEnvironment() || truthyEnv("JELLYCAT_SEED_DEFAULT_CATALOG")
}

func truthyEnv(name string) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func personalizePlayerForTeam(player models.Player, team models.Team) models.Player {
	seed := team.ID + "|" + team.Name + "|" + player.ID
	hash := deterministicHash(seed)
	player.Points = clampInt(player.Points+int(hash%241)-120, 80, 500)
	player.CuddlePoints = clampInt(player.CuddlePoints+int((hash>>8)%61)-30, 10, 100)
	return player
}

func deterministicHash(value string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return hasher.Sum32()
}

// randomCuddlePoints generates a random cuddle points value between 25 and 79 (inclusive)
// using crypto/rand for thread safety
func randomCuddlePoints() int {
	// Generate a random number between 0 and 54 (79-25=54)
	n, err := rand.Int(rand.Reader, big.NewInt(55))
	if err != nil {
		// Double-fallback: return the middle of the range to avoid modulo bias
		return 50
	}
	return int(n.Int64()) + 25
}
