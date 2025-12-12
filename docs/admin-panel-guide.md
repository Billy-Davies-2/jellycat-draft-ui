# Admin Panel Guide

This guide explains the Admin Panel features and how to use them to manage the Jellycat Draft application.

## Table of Contents

- [Access Control](#access-control)
- [Features](#features)
- [Adding Jellycats](#adding-jellycats)
- [Managing Cuddle Points](#managing-cuddle-points)
- [Managing Draft Scores](#managing-draft-scores)
- [Draft Controls](#draft-controls)

## Access Control

The Admin Panel is **restricted to administrators only** and requires:

1. **Authentication**: User must be logged in via Authentik OAuth2
2. **Authorization**: User must be in the `admins` group in Authentik

### Setting Up Admin Access

In your Authentik instance:

1. Navigate to **Directory** → **Groups**
2. Create or edit the `admins` group
3. Add users who should have admin access
4. Users in this group will see the Admin Panel link in the navigation

### Access URL

```
https://your-domain.com/admin
```

Non-admin users who try to access this URL will receive:
```
HTTP 403 Forbidden: Admin access required
```

## Features

The Admin Panel provides three main sections:

### 1. Add New Jellycat
Add new Jellycat plush toys to the draft pool

### 2. Current Jellycats
View and manage all Jellycats in the database

### 3. Draft Controls
Reset the draft and manage draft settings

## Adding Jellycats

### Form Fields

| Field | Description | Options | Required |
|-------|-------------|---------|----------|
| **Name** | Jellycat's name | Text input | ✅ |
| **Position** | Player position | CC, SS, HH, CH | ✅ |
| **Tier** | Rarity tier | S, A, B, C | ✅ |
| **Team** | Category/team | e.g., Woodland, Safari | ✅ |
| **Cuddle Points** | Initial points | Number (0+) | ✅ |
| **Image URL** | Path to image | e.g., /static/images/name.png | Optional |

### Position Types

- **CC (Cuddle Companion)**: Primary cuddling role
- **SS (Snuggle Specialist)**: Expert snuggling
- **HH (Hug Hero)**: Heroic hugging abilities
- **CH (Comfort Helper)**: Supportive comfort role

### Tier System

- **S (Supreme)**: Legendary Jellycats - highest rarity
- **A (Amazing)**: Exceptional Jellycats
- **B (Brilliant)**: Great Jellycats
- **C (Charming)**: Good Jellycats

### Example: Adding a New Jellycat

```
Name: Bashful Bunny
Position: CC (Cuddle Companion)
Tier: A (Amazing)
Team: Woodland
Cuddle Points: 65
Image URL: /static/images/bashful-bunny.png
```

Click **➕ Add Jellycat** to save.

### What Happens When You Add a Jellycat

1. Jellycat is saved to the database with a unique ID
2. Event `players:add` is published to NATS
3. All connected clients receive real-time update
4. Jellycat appears in the draft pool (if not yet drafted)
5. Jellycat is immediately available for drafting

## Managing Cuddle Points

### Viewing Current Points

In the **Current Jellycats** section, each Jellycat card displays:
- Current cuddle points as a badge
- Visual indicators (colored borders)
- Draft status (drafted or available)

### Modifying Points via API

Use the API endpoint to update points programmatically:

```bash
# HTTP API
curl -X POST https://your-domain.com/api/players/points \
  -H "Content-Type: application/json" \
  -d '{
    "id": "jellycat-id",
    "points": 75
  }'
```

```go
// gRPC API
client.SetPlayerPoints(ctx, &pb.SetPlayerPointsRequest{
    Id: "jellycat-id",
    Points: 75,
})
```

### Points Updates

When points are updated:
1. New points are saved to database
2. Event `players:updatePoints` is published
3. All clients receive real-time update
4. Updated points appear immediately in UI

### Default Points

From the codebase memory:
- **Default cuddle points**: 50
- **Random assignment**: 25-79 when admin adds player without explicit value
- **Draft adjustments**: Points are adjusted based on draft pick number

## Managing Draft Scores

### Draft Pick Adjustments

Based on the repository memory, cuddle points are automatically adjusted during the draft:

| Draft Pick | Points Adjustment | Formula |
|------------|-------------------|---------|
| 1-6 | **Gain points** | `20 - (pick × 2)` |
| 7-12 | **No change** | `0` |
| 13-18 | **Lose points** | `8 - pick` |

**Examples**:
- Pick #1: +18 points (20 - 2 = 18)
- Pick #3: +14 points (20 - 6 = 14)
- Pick #6: +8 points (20 - 12 = 8)
- Pick #10: No change
- Pick #15: -7 points (8 - 15 = -7)
- Pick #18: -10 points (8 - 18 = -10)

These adjustments happen automatically when a player is drafted.

### Viewing Draft Scores

On the main draft page (`/draft`), each team's total score is calculated as:
```
Total Score = Sum of all drafted players' cuddle points
```

Teams are ranked by total score on the leaderboard.

## Draft Controls

### Reset Draft

⚠️ **Warning**: This action cannot be undone!

Clicking **🔄 Reset Draft** will:
1. Show confirmation dialog
2. Clear all draft picks
3. Restore all players to available status
4. Reset teams to default configuration
5. Clear all chat messages (optional)
6. Reload the page automatically

### When to Reset

- Starting a new draft season
- Testing draft functionality
- Fixing draft mistakes
- Running a practice draft

### What is Preserved

✅ **Preserved**:
- Player data (names, images, positions)
- Team configurations
- User accounts and admin access

❌ **Cleared**:
- Draft picks
- Player "drafted" status
- Chat history
- Draft order

## Current Jellycats Section

### Card Information

Each Jellycat card displays:

1. **Image**: Visual preview (with fallback to placeholder)
2. **Name**: Jellycat's name
3. **Team**: Category (Woodland, Safari, etc.)
4. **Position Badge**: CC, SS, HH, or CH (color-coded)
5. **Tier Badge**: S, A, B, or C (color-coded)
6. **Points Badge**: Current cuddle points
7. **Draft Status**: Shows which team drafted them (if drafted)

### Visual Indicators

**Position Colors**:
- CC: Pink background
- SS: Purple background
- HH: Blue background
- CH: Green background

**Tier Colors**:
- S: Gold background (Supreme)
- A: Green background (Amazing)
- B: Blue background (Brilliant)
- C: Gray background (Charming)

**Draft Status**:
- Available: White background, full opacity
- Drafted: Gray background, reduced opacity, shows team name

### Filtering (Future Enhancement)

Future versions may include:
- Filter by position
- Filter by tier
- Filter by draft status
- Search by name
- Sort by points

## Best Practices

### Adding Jellycats

1. **Consistent naming**: Use official Jellycat names
2. **Accurate points**: Base on actual Jellycat popularity/rarity
3. **Quality images**: Use clear, consistent image sizes
4. **Proper categorization**: Assign correct positions and tiers
5. **Team consistency**: Use standard team names (Woodland, Safari, Ocean, etc.)

### Managing Points

1. **Document changes**: Keep notes on why points were adjusted
2. **Gradual adjustments**: Avoid drastic point changes
3. **Balance consideration**: Ensure draft remains competitive
4. **Transparency**: Communicate point changes to players

### Draft Management

1. **Backup before reset**: Consider exporting data before major resets
2. **Communicate resets**: Warn users before resetting an active draft
3. **Test in development**: Use development environment for testing
4. **Regular maintenance**: Periodically review and update player data

## API Access for Admins

Admins can use API endpoints for bulk operations:

### Add Multiple Players

```bash
# Add players in batch (loop through list)
for jellycat in jellycats.json; do
  curl -X POST https://your-domain.com/api/players/add \
    -H "Content-Type: application/json" \
    -d @jellycat.json
done
```

### Export Player Data

```bash
# Get all players
curl https://your-domain.com/api/draft/state | jq '.players'
```

### Update Points in Bulk

```bash
# Update multiple players' points
while IFS=, read -r id points; do
  curl -X POST https://your-domain.com/api/players/points \
    -H "Content-Type: application/json" \
    -d "{\"id\":\"$id\",\"points\":$points}"
done < points.csv
```

## Troubleshooting

### "Forbidden: Admin access required"

**Solution**: Ensure you are in the `admins` group in Authentik:
1. Log in to Authentik admin panel
2. Navigate to Directory → Groups
3. Check if your user is in `admins` group
4. Add yourself if missing
5. Log out and log back in to refresh permissions

### Changes Not Appearing

**Solution**: 
1. Check browser console for errors
2. Verify Server-Sent Events (SSE) connection is active
3. Refresh the page
4. Check application logs for errors

### Image Not Loading

**Solution**:
1. Verify image exists at specified path
2. Check image URL format: `/static/images/filename.png`
3. Ensure images are included in deployment
4. Use placeholder if image is missing (automatic fallback)

## Security Notes

- Admin panel requires authentication AND authorization
- All admin actions are logged with timestamps
- API endpoints validate admin privileges
- Changes are published to all connected clients
- Database changes are transactional

## References

- [Authentication Setup](../AUTH_SETUP.md)
- [Main README](../README.md)
- [API Documentation](../README.md#api-endpoints)
- [Deployment Guide](../README.md#deployment)
