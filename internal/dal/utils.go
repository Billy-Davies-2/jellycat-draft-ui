package dal

import (
	"crypto/rand"
	"math/big"
)

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
