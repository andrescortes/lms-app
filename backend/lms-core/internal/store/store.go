package store

import (
	"context"

	"lms-app/backend/lms-core/internal/model"
)

// LevelStoreInterface defines the contract for level storage operations.
// This allows for mocking in tests and decouples the transport layer from concrete implementations.
type LevelStoreInterface interface {
	ListLevels(ctx context.Context) ([]model.CEFRLevel, error)
}

// Ensure LevelStore implements LevelStoreInterface
var _ LevelStoreInterface = (*LevelStore)(nil)