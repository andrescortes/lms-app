package transport

import (
	"log/slog"
	"net/http"

	"lms-app/backend/lms-core/internal/model"
	"lms-app/backend/lms-core/internal/store"
)

type LevelHandler struct {
	levelStore store.LevelStoreInterface
}

func NewLevelHandler(levelStore store.LevelStoreInterface) *LevelHandler {
	return &LevelHandler{
		levelStore: levelStore,
	}
}

// GetLevels handle GET /api/lms/levels.
func (h *LevelHandler) GetLevels(w http.ResponseWriter, r *http.Request) {
	user, ok := GetTrustedUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	levels, err := h.levelStore.ListLevels(r.Context())
	if err != nil {
		slog.Error("error listing CEFR levels", "error", err, "user_id", user.ID)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Determine numeric rank level currently from student (by default 1 = A1)
	currentRank := 1
	for _, lvl := range levels {
		if lvl.Code == user.CEFRLevel {
			currentRank = lvl.RankOrder
			break
		}
	}

	// Enrich each level with its custom state for the student
	for i := range levels {
		levels[i].IsCurrent = levels[i].Code == user.CEFRLevel
		levels[i].Unlocked = levels[i].RankOrder <= currentRank
	}

	writeJSON(w, http.StatusOK, model.LevelsResponse{
		UserID:       user.ID,
		CurrentLevel: user.CEFRLevel,
		Levels:       levels,
	})
}
