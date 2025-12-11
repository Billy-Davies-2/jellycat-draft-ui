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
		// Fallback: try reading random bytes directly
		var b [1]byte
		_, readErr := rand.Read(b[:])
		if readErr != nil {
			// Final fallback: return the middle of the range
			return 50
		}
		// Use the random byte to generate a value in range
		return int(b[0]%55) + 25
	}
	return int(n.Int64()) + 25
}
