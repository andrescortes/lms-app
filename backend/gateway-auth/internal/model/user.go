package model

import (
	"errors"
	"time"
)

var (
	ErrEmailAlreadyExists = errors.New("email is already registered")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidInput       = errors.New("invalid input data")
)

var ValidCEFRLevels = map[string]bool{
	"A1": true,
	"A2": true,
	"B1": true,
	"B2": true,
	"C1": true,
	"C2": true,
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         string    `json:"role"`
	CEFRLevel    string    `json:"cefr_level"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FullName  string `json:"full_name"`
	CEFRLevel string `json:"cefr_level,omitempty"`
}
