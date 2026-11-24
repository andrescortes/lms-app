package store

import (
	"context"
	"database/sql"
	"fmt"

	"lms-app/backend/lms-core/internal/model"
)

type LevelStore struct {
	db *sql.DB
}

func NewLevelStore(db *sql.DB) *LevelStore {
	return &LevelStore{db: db}
}

func (s *LevelStore) ListLevels(ctx context.Context) ([]model.CEFRLevel, error) {
	const query = `
		SELECT code, name, description, rank_order
		FROM cefr_levels
		ORDER BY rank_order ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying cefr levels: %w", err)
	}
	defer rows.Close()

	var levels []model.CEFRLevel
	for rows.Next() {
		var lvl model.CEFRLevel
		if err := rows.Scan(&lvl.Code, &lvl.Name, &lvl.Description, &lvl.RankOrder); err != nil {
			return nil, fmt.Errorf("scanning cefr level row: %w", err)
		}
		levels = append(levels, lvl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cefr level rows: %w", err)
	}

	return levels, nil
}
