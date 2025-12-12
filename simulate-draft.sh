#!/bin/bash
# Script to simulate a draft for local development and testing
# This script demonstrates the cuddle points system by simulating a complete draft

set -e

echo "🎯 Jellycat Draft Simulator"
echo "=========================="
echo ""
echo "This script simulates a draft to demonstrate the cuddle points system:"
echo "- Players start with cuddle points of 50 (or random 25-79 if added by admin)"
echo "- Early picks (1-6) gain bonus cuddle points (+8 to +18)"
echo "- Late picks (13-18) lose cuddle points (-5 to -10)"
echo ""

# Check if the application is running
if ! curl -s http://localhost:3000/api/state > /dev/null 2>&1; then
    echo "❌ Error: Jellycat draft application is not running on http://localhost:3000"
    echo ""
    echo "Please start the application first:"
    echo "  DB_DRIVER=sqlite SQLITE_FILE=dev.sqlite ./jellycat-draft"
    echo ""
    exit 1
fi

echo "✅ Application is running on http://localhost:3000"
echo ""

# Reset the draft to start fresh
echo "🔄 Resetting draft..."
curl -s -X POST http://localhost:3000/api/reset > /dev/null
echo "✅ Draft reset complete"
echo ""

# Get the initial state to see teams and players
echo "📋 Fetching teams and players..."
STATE=$(curl -s http://localhost:3000/api/state)
TEAMS=$(echo "$STATE" | jq -r '.teams[].id' | head -6)
PLAYERS=$(echo "$STATE" | jq -r '.players[].id' | head -18)

TEAM_COUNT=$(echo "$TEAMS" | wc -l)
PLAYER_COUNT=$(echo "$PLAYERS" | wc -l)

echo "✅ Found $TEAM_COUNT teams and $PLAYER_COUNT players"
echo ""

# Simulate drafting all 18 players
echo "🎲 Simulating draft picks..."
echo ""

PICK_NUMBER=1
TEAM_ARRAY=($TEAMS)
PLAYER_ARRAY=($PLAYERS)

for PLAYER_ID in "${PLAYER_ARRAY[@]}"; do
    # Round-robin team selection
    TEAM_INDEX=$(( (PICK_NUMBER - 1) % TEAM_COUNT ))
    TEAM_ID="${TEAM_ARRAY[$TEAM_INDEX]}"
    
    # Get player info before draft
    BEFORE=$(curl -s http://localhost:3000/api/state | jq -r ".players[] | select(.id == \"$PLAYER_ID\")")
    PLAYER_NAME=$(echo "$BEFORE" | jq -r '.name')
    CUDDLE_BEFORE=$(echo "$BEFORE" | jq -r '.cuddlePoints')
    
    # Draft the player
    RESULT=$(curl -s -X POST http://localhost:3000/api/draft/pick \
        -H "Content-Type: application/json" \
        -d "{\"playerId\":\"$PLAYER_ID\",\"teamId\":\"$TEAM_ID\"}")
    
    # Get player info after draft
    AFTER=$(curl -s http://localhost:3000/api/state | jq -r ".players[] | select(.id == \"$PLAYER_ID\")")
    CUDDLE_AFTER=$(echo "$AFTER" | jq -r '.cuddlePoints')
    TEAM_NAME=$(echo "$AFTER" | jq -r '.draftedBy')
    
    # Calculate adjustment
    ADJUSTMENT=$((CUDDLE_AFTER - CUDDLE_BEFORE))
    
    # Determine adjustment type
    if [ "$ADJUSTMENT" -gt 0 ]; then
        ADJUSTMENT_TEXT="+$ADJUSTMENT"
        EMOJI="📈"
    elif [ "$ADJUSTMENT" -lt 0 ]; then
        ADJUSTMENT_TEXT="$ADJUSTMENT"
        EMOJI="📉"
    else
        ADJUSTMENT_TEXT="±0"
        EMOJI="➡️"
    fi
    
    printf "Pick #%-2d: %-25s → %-20s | Cuddle: %2d → %2d (%s) %s\n" \
        "$PICK_NUMBER" "$PLAYER_NAME" "$TEAM_NAME" \
        "$CUDDLE_BEFORE" "$CUDDLE_AFTER" "$ADJUSTMENT_TEXT" "$EMOJI"
    
    PICK_NUMBER=$((PICK_NUMBER + 1))
done

echo ""
echo "✅ Draft simulation complete!"
echo ""
echo "📊 Cuddle Points Summary:"
echo "  • Picks 1-6:  Early picks gained +8 to +18 points 📈"
echo "  • Picks 7-12: Mid picks had no adjustment ➡️"
echo "  • Picks 13-18: Late picks lost -5 to -10 points 📉"
echo ""
echo "View the full draft state at: http://localhost:3000"
echo ""
