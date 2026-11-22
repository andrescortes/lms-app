package service

import (
	"context"
	"fmt"
	"lms-app/backend/gateway-auth/internal/model"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
}

type AuthService struct {
	users UserRepository
}

func NewAuthService(users UserRepository) *AuthService {
	return &AuthService{users: users}
}

func (s *AuthService) RegisterUser(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	fullName := strings.TrimSpace(req.FullName)
	cefrLevel := strings.ToUpper(strings.TrimSpace(req.CEFRLevel))

	if email == "" {
		return nil, fmt.Errorf("%w: email is required", model.ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, fmt.Errorf("%w: invalid email format", model.ErrInvalidInput)
	}

	if len(fullName) < 2 {
		return nil, fmt.Errorf("%w: full_name must have at least 2 characters", model.ErrInvalidInput)
	}
	if len(req.Password) < 8 {
		return nil, fmt.Errorf("%w: password must have at least 8 characters", model.ErrInvalidInput)
	}
	if len(req.Password) > 72 {
		return nil, fmt.Errorf("%w: password cannot exceed 72 characters", model.ErrInvalidInput)
	}

	if cefrLevel == "" {
		cefrLevel = "A1"
	} else if !model.ValidCEFRLevels[cefrLevel] {
		return nil, fmt.Errorf("%w: cefr_level must be one of A1, A2, B1, B2, C1, C2", model.ErrInvalidInput)
	}

	// 5. Generar hash bcrypt de la contraseña
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hashedBytes),
		FullName:     fullName,
		Role:         "student",
		CEFRLevel:    cefrLevel,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.users.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
