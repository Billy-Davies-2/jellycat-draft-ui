package mocks

import (
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/dal"
	"github.com/Billy-Davies-2/jellycat-draft-ui/internal/logger"
)

// MockPostgresDAL provides a mock Postgres implementation using SQLite for local development
type MockPostgresDAL struct {
	dal.DraftDAL
}

// NewMockPostgresDAL creates a mock Postgres DAL using SQLite
func NewMockPostgresDAL(sqliteFile string) (*MockPostgresDAL, error) {
	logger.Info("Using MOCK Postgres (SQLite) for local development")

	sqliteDAL, err := dal.NewSQLiteDAL(sqliteFile)
	if err != nil {
		return nil, err
	}

	return &MockPostgresDAL{
		DraftDAL: sqliteDAL,
	}, nil
}
