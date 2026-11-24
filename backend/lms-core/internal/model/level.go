package model

type CEFRLevel struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RankOrder   int    `json:"rank_order"`
	Unlocked    bool   `json:"unlocked"`
	IsCurrent   bool   `json:"is_current"`
}

type LevelsResponse struct {
	UserID       string      `json:"user_id"`
	CurrentLevel string      `json:"current_level"`
	Levels       []CEFRLevel `json:"levels"`
}
