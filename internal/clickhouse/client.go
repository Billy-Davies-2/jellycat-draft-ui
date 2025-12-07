package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client provides ClickHouse integration for cuddle points
type Client struct {
	conn driver.Conn
}

// NewClient creates a new ClickHouse client
func NewClient(addr, database, username, password string) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	return &Client{conn: conn}, nil
}

// GetCuddlePoints retrieves cuddle points for a Jellycat from ClickHouse
// This queries aggregated metrics to calculate cuddle points
func (c *Client) GetCuddlePoints(jellycatID string) (int, error) {
	var points int

	query := `
		SELECT 
			toInt32(
				countDistinct(user_id) * 10 +  -- Unique users who interacted
				count() / 10 +                   -- Total interactions
				sum(duration) / 60               -- Time spent (seconds to minutes)
			) as cuddle_points
		FROM jellycat_interactions
		WHERE jellycat_id = $1
		AND timestamp >= now() - INTERVAL 30 DAY
	`

	row := c.conn.QueryRow(context.Background(), query, jellycatID)
	if err := row.Scan(&points); err != nil {
		return 0, err
	}

	return points, nil
}

// GetAllCuddlePoints retrieves cuddle points for all Jellycats
func (c *Client) GetAllCuddlePoints() (map[string]int, error) {
	points := make(map[string]int)

	query := `
		SELECT 
			jellycat_id,
			toInt32(
				countDistinct(user_id) * 10 +
				count() / 10 +
				sum(duration) / 60
			) as cuddle_points
		FROM jellycat_interactions
		WHERE timestamp >= now() - INTERVAL 30 DAY
		GROUP BY jellycat_id
	`

	rows, err := c.conn.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		var pts int
		if err := rows.Scan(&id, &pts); err != nil {
			return nil, err
		}
		points[id] = pts
	}

	return points, nil
}

// SyncCuddlePoints updates player cuddle points from ClickHouse
// This should be called periodically to keep points up-to-date
func (c *Client) SyncCuddlePoints(updateFunc func(playerID string, points int) error) error {
	allPoints, err := c.GetAllCuddlePoints()
	if err != nil {
		return err
	}

	for playerID, points := range allPoints {
		if err := updateFunc(playerID, points); err != nil {
			return fmt.Errorf("failed to update points for %s: %w", playerID, err)
		}
	}

	return nil
}

// Close closes the ClickHouse connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
