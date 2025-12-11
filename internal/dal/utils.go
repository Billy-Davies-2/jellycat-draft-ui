package dal

import (
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"sync"
)

var (
	// Thread-safe random number generator for cuddle points
	rngMutex sync.Mutex
)

// randomCuddlePoints generates a random cuddle points value between 25 and 79 (inclusive)
// using crypto/rand for thread safety
func randomCuddlePoints() int {
	// Generate a random number between 0 and 54 (79-25=54)
	n, err := rand.Int(rand.Reader, big.NewInt(55))
	if err != nil {
		// Fallback to a simpler method if crypto/rand fails
		rngMutex.Lock()
		defer rngMutex.Unlock()
		var b [8]byte
		rand.Read(b[:])
		val := binary.LittleEndian.Uint64(b[:])
		return int(val%55) + 25
	}
	return int(n.Int64()) + 25
}
