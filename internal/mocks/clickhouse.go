package mocks

import (
	"log"
	"math/rand"
)

// MockClickHouseClient provides a mock ClickHouse client for local development
type MockClickHouseClient struct {
	basePoints map[string]int
}

// NewMockClickHouseClient creates a mock ClickHouse client
func NewMockClickHouseClient() *MockClickHouseClient {
	log.Println("Using MOCK ClickHouse client for local development")

	return &MockClickHouseClient{
		basePoints: map[string]int{
			"1":  324, // Bashful Bunny
			"2":  298, // Fuddlewuddle Lion
			"3":  287, // Cordy Roy Elephant
			"4":  251, // Blossom Tulip Bunny
			"5":  312, // Amuseable Avocado
			"6":  276, // Octopus Ollie
			"7":  268, // Jellycat Dragon
			"8":  245, // Bashful Lamb
			"9":  289, // Amuseable Pineapple
			"10": 234, // Cordy Roy Fox
			"11": 256, // Blossom Peach Bunny
			"12": 267, // Amuseable Taco
			"13": 278, // Bashful Unicorn
			"14": 243, // Jellycat Penguin
			"15": 229, // Amuseable Moon
			"16": 241, // Cordy Roy Pig
			"17": 235, // Bashful Tiger
			"18": 228, // Amuseable Donut
		},
	}
}

// GetCuddlePoints returns mock cuddle points with slight variation
func (m *MockClickHouseClient) GetCuddlePoints(jellycatID string) (int, error) {
	base, ok := m.basePoints[jellycatID]
	if !ok {
		base = 200 // Default for unknown jellycats
	}

	// Add some randomness for realism (±10%)
	variance := rand.Intn(int(float64(base)*0.2)) - int(float64(base)*0.1)
	return base + variance, nil
}

// GetAllCuddlePoints returns all mock cuddle points
func (m *MockClickHouseClient) GetAllCuddlePoints() (map[string]int, error) {
	result := make(map[string]int)
	for id, base := range m.basePoints {
		variance := rand.Intn(int(float64(base)*0.2)) - int(float64(base)*0.1)
		result[id] = base + variance
	}
	return result, nil
}

// SyncCuddlePoints updates player cuddle points (mock implementation)
func (m *MockClickHouseClient) SyncCuddlePoints(updateFunc func(playerID string, points int) error) error {
	allPoints, err := m.GetAllCuddlePoints()
	if err != nil {
		return err
	}

	for playerID, points := range allPoints {
		if err := updateFunc(playerID, points); err != nil {
			log.Printf("Failed to update points for %s: %v", playerID, err)
		}
	}

	log.Println("Mock ClickHouse: Synced cuddle points for all Jellycats")
	return nil
}

// Close is a no-op for mock client
func (m *MockClickHouseClient) Close() error {
	return nil
}
