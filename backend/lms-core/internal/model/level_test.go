package model

import (
	"encoding/json"
	"testing"
)

func TestCEFRLevel_JSONMarshaling(t *testing.T) {
	level := CEFRLevel{
		Code:        "B1",
		Name:        "Intermediate",
		Description: "Can deal with most situations",
		RankOrder:   3,
		Unlocked:    true,
		IsCurrent:   false,
	}

	data, err := json.Marshal(level)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded CEFRLevel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Code != level.Code {
		t.Errorf("Code: expected %q, got %q", level.Code, decoded.Code)
	}
	if decoded.Name != level.Name {
		t.Errorf("Name: expected %q, got %q", level.Name, decoded.Name)
	}
	if decoded.Description != level.Description {
		t.Errorf("Description: expected %q, got %q", level.Description, decoded.Description)
	}
	if decoded.RankOrder != level.RankOrder {
		t.Errorf("RankOrder: expected %d, got %d", level.RankOrder, decoded.RankOrder)
	}
	if decoded.Unlocked != level.Unlocked {
		t.Errorf("Unlocked: expected %v, got %v", level.Unlocked, decoded.Unlocked)
	}
	if decoded.IsCurrent != level.IsCurrent {
		t.Errorf("IsCurrent: expected %v, got %v", level.IsCurrent, decoded.IsCurrent)
	}
}

func TestCEFRLevel_JSONUnmarshaling(t *testing.T) {
	jsonData := `{
		"code": "C1",
		"name": "Advanced",
		"description": "Can express ideas fluently",
		"rank_order": 5,
		"unlocked": false,
		"is_current": true
	}`

	var level CEFRLevel
	if err := json.Unmarshal([]byte(jsonData), &level); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if level.Code != "C1" {
		t.Errorf("expected Code 'C1', got '%s'", level.Code)
	}
	if level.Name != "Advanced" {
		t.Errorf("expected Name 'Advanced', got '%s'", level.Name)
	}
	if level.Description != "Can express ideas fluently" {
		t.Errorf("expected Description 'Can express ideas fluently', got '%s'", level.Description)
	}
	if level.RankOrder != 5 {
		t.Errorf("expected RankOrder 5, got %d", level.RankOrder)
	}
	if level.Unlocked != false {
		t.Errorf("expected Unlocked false, got %v", level.Unlocked)
	}
	if level.IsCurrent != true {
		t.Errorf("expected IsCurrent true, got %v", level.IsCurrent)
	}
}

func TestCEFRLevel_EmptyValues(t *testing.T) {
	var level CEFRLevel

	data, err := json.Marshal(level)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded CEFRLevel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.Code != "" {
		t.Errorf("expected empty Code, got %q", decoded.Code)
	}
	if decoded.RankOrder != 0 {
		t.Errorf("expected RankOrder 0, got %d", decoded.RankOrder)
	}
	if decoded.Unlocked != false {
		t.Errorf("expected Unlocked false, got %v", decoded.Unlocked)
	}
	if decoded.IsCurrent != false {
		t.Errorf("expected IsCurrent false, got %v", decoded.IsCurrent)
	}
}

func TestLevelsResponse_JSONMarshaling(t *testing.T) {
	response := LevelsResponse{
		UserID:       "user-123",
		CurrentLevel: "B2",
		Levels: []CEFRLevel{
			{Code: "A1", Name: "Beginner", RankOrder: 1, Unlocked: true, IsCurrent: false},
			{Code: "A2", Name: "Elementary", RankOrder: 2, Unlocked: true, IsCurrent: false},
			{Code: "B1", Name: "Intermediate", RankOrder: 3, Unlocked: true, IsCurrent: false},
			{Code: "B2", Name: "Upper Intermediate", RankOrder: 4, Unlocked: true, IsCurrent: true},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded LevelsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded.UserID != response.UserID {
		t.Errorf("UserID: expected %q, got %q", response.UserID, decoded.UserID)
	}
	if decoded.CurrentLevel != response.CurrentLevel {
		t.Errorf("CurrentLevel: expected %q, got %q", response.CurrentLevel, decoded.CurrentLevel)
	}
	if len(decoded.Levels) != len(response.Levels) {
		t.Errorf("Levels length: expected %d, got %d", len(response.Levels), len(decoded.Levels))
	}

	for i, level := range response.Levels {
		if decoded.Levels[i].Code != level.Code {
			t.Errorf("Level[%d].Code: expected %q, got %q", i, level.Code, decoded.Levels[i].Code)
		}
	}
}

func TestLevelsResponse_EmptyLevels(t *testing.T) {
	response := LevelsResponse{
		UserID:       "user-123",
		CurrentLevel: "A1",
		Levels:       []CEFRLevel{},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded LevelsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(decoded.Levels) != 0 {
		t.Errorf("expected 0 levels, got %d", len(decoded.Levels))
	}
}