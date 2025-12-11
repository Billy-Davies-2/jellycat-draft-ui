package dal

import (
	"fmt"
	"testing"

	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/models"
)

// TestMemoryDALDraftCuddlePointsAdjustment tests that cuddle points are adjusted based on draft order
func TestMemoryDALDraftCuddlePointsAdjustment(t *testing.T) {
dal := NewMemoryDAL()

// Get teams
state, err := dal.GetState()
if err != nil {
t.Fatalf("GetState() failed: %v", err)
}

if len(state.Teams) < 3 {
t.Fatalf("Need at least 3 teams, got %d", len(state.Teams))
}

// Add 18 players with known cuddle points
initialCuddle := 50
playerIDs := []string{}

for i := 1; i <= 18; i++ {
player := &models.Player{
Name:         fmt.Sprintf("Test Player %d", i),
Position:     "CC",
Team:         "Test",
Points:       100,
CuddlePoints: initialCuddle,
Tier:         models.TierA,
}

addedPlayer, err := dal.AddPlayer(player)
if err != nil {
t.Fatalf("AddPlayer() failed for player %d: %v", i, err)
}
playerIDs = append(playerIDs, addedPlayer.ID)
}

// Now draft all players in sequence
for i, playerID := range playerIDs {
teamIdx := i % len(state.Teams)
err = dal.DraftPlayer(playerID, state.Teams[teamIdx].ID)
if err != nil {
t.Fatalf("DraftPlayer() failed for player %d: %v", i+1, err)
}
}

// Get final state
state, err = dal.GetState()
if err != nil {
t.Fatalf("GetState() failed: %v", err)
}

// Test specific draft picks
testCases := []struct {
pickNumber int
playerID   string
expectedAdjust int
description string
}{
{1, playerIDs[0], 18, "Pick 1 should get +18"},
{2, playerIDs[1], 16, "Pick 2 should get +16"},
{6, playerIDs[5], 8, "Pick 6 should get +8"},
{7, playerIDs[6], 0, "Pick 7 should get 0"},
{12, playerIDs[11], 0, "Pick 12 should get 0"},
{13, playerIDs[12], -5, "Pick 13 should get -5"},
{18, playerIDs[17], -10, "Pick 18 should get -10"},
}

for _, tc := range testCases {
found := false
for _, p := range state.Players {
if p.ID == tc.playerID && p.Drafted {
found = true
expectedCuddle := initialCuddle + tc.expectedAdjust
if expectedCuddle < 10 {
expectedCuddle = 10
}
if expectedCuddle > 100 {
expectedCuddle = 100
}

if p.CuddlePoints != expectedCuddle {
t.Errorf("%s: pick #%d expected cuddle_points=%d, got %d", 
tc.description, tc.pickNumber, expectedCuddle, p.CuddlePoints)
}
break
}
}

if !found {
t.Errorf("Player for pick #%d not found in drafted state", tc.pickNumber)
}
}
}

// TestMemoryDALEarlyPicksBonusCuddlePoints tests early draft picks get bonus cuddle points
func TestMemoryDALEarlyPicksBonusCuddlePoints(t *testing.T) {
dal := NewMemoryDAL()

state, _ := dal.GetState()
if len(state.Teams) == 0 {
t.Fatal("No teams available")
}

// Add a player with 50 cuddle points
player := &models.Player{
Name:         "Early Pick Test",
Position:     "CC",
Team:         "Test",
Points:       100,
CuddlePoints: 50,
Tier:         models.TierS,
}

addedPlayer, err := dal.AddPlayer(player)
if err != nil {
t.Fatalf("AddPlayer() failed: %v", err)
}

// Draft as first overall pick
err = dal.DraftPlayer(addedPlayer.ID, state.Teams[0].ID)
if err != nil {
t.Fatalf("DraftPlayer() failed: %v", err)
}

// Get updated state
state, _ = dal.GetState()

// Find the player and verify cuddle points increased
for _, p := range state.Players {
if p.ID == addedPlayer.ID {
if p.CuddlePoints <= 50 {
t.Errorf("First overall pick should have increased cuddle points, got %d", p.CuddlePoints)
}
// First pick should get +18 bonus (50 + 18 = 68)
if p.CuddlePoints != 68 {
t.Errorf("First overall pick should have 68 cuddle points, got %d", p.CuddlePoints)
}
break
}
}
}

// TestMemoryDALLatePicksLoseCuddlePoints tests late draft picks lose cuddle points
func TestMemoryDALLatePicksLoseCuddlePoints(t *testing.T) {
dal := NewMemoryDAL()

state, _ := dal.GetState()
if len(state.Teams) == 0 {
t.Fatal("No teams available")
}

// Draft 17 dummy players first to make the next one pick #18
for i := 0; i < 17; i++ {
dummyPlayer := &models.Player{
Name:         fmt.Sprintf("Dummy %d", i),
Position:     "CC",
Team:         "Dummy",
Points:       50,
CuddlePoints: 50,
Tier:         models.TierB,
}

p, err := dal.AddPlayer(dummyPlayer)
if err != nil {
t.Fatalf("Failed to add dummy player %d: %v", i, err)
}

teamIdx := i % len(state.Teams)
err = dal.DraftPlayer(p.ID, state.Teams[teamIdx].ID)
if err != nil {
t.Fatalf("Failed to draft dummy player %d: %v", i, err)
}
}

// Now add the player we want to test (this will be pick #18)
player := &models.Player{
Name:         "Late Pick Test",
Position:     "CC",
Team:         "Test",
Points:       100,
CuddlePoints: 50,
Tier:         models.TierS,
}

addedPlayer, err := dal.AddPlayer(player)
if err != nil {
t.Fatalf("AddPlayer() failed: %v", err)
}

// Draft as 18th overall pick
teamIdx := 17 % len(state.Teams)
err = dal.DraftPlayer(addedPlayer.ID, state.Teams[teamIdx].ID)
if err != nil {
t.Fatalf("DraftPlayer() failed: %v", err)
}

// Get updated state
state, _ = dal.GetState()

// Find the player and verify cuddle points decreased
for _, p := range state.Players {
if p.ID == addedPlayer.ID {
if p.CuddlePoints >= 50 {
t.Errorf("18th overall pick should have decreased cuddle points, got %d", p.CuddlePoints)
}
// 18th pick should get -10 penalty (50 - 10 = 40)
if p.CuddlePoints != 40 {
t.Errorf("18th overall pick should have 40 cuddle points, got %d", p.CuddlePoints)
}
break
}
}
}
