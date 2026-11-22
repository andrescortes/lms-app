package service

import (
	"context"
	"errors"
	"fmt"
	"lms-app/backend/gateway-auth/internal/model"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserById(ctx context.Context, id string) (*model.User, error)
	SaveRefreshToken(ctx context.Context, rt *model.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RotateRefreshToken(ctx context.Context, oldTokenID string, newToken *model.RefreshToken) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error
}

type AuthService struct {
	authRepo   AuthRepository
	jwtService *JWTService
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewAuthService(
	authRepo AuthRepository,
	jwtService *JWTService,
	accessTTL time.Duration,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		authRepo:   authRepo,
		jwtService: jwtService,
		accessTTL:  accessTTL,
		refreshTTL: accessTTL,
	}
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

	if err := s.authRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req model.LoginRequest) (*model.AuthResponse, string, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		return nil, "", fmt.Errorf("%w: email and password are required", model.ErrInvalidInput)
	}

	user, err := s.authRepo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrUserNotFound) {
			return nil, "", model.ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, "", model.ErrInvalidCredentials
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, "", err
	}

	rawRefresh, refreshHash, err := s.jwtService.GenerateOpaqueRefreshToken()
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	rt := &model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: refreshHash,
		ExpiresAt: now.Add(s.refreshTTL),
		Revoked:   false,
		CreatedAt: now,
	}

	if err := s.authRepo.SaveRefreshToken(ctx, rt); err != nil {
		return nil, "", err
	}

	return &model.AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTTL.Seconds()),
		User:        user,
	}, rawRefresh, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*model.AuthResponse, string, error) {
	if strings.TrimSpace(rawRefreshToken) == "" {
		return nil, "", model.ErrInvalidRefreshToken
	}

	tokenHash := HashRefreshToken(rawRefreshToken)
	storedToken, err := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, "", err
	}

	if storedToken.Revoked {
		_ = s.authRepo.RevokeAllUserRefreshTokens(ctx, storedToken.UserID)
		return nil, "", model.ErrInvalidRefreshToken
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return nil, "", model.ErrInvalidRefreshToken
	}

	user, err := s.authRepo.GetUserById(ctx, storedToken.UserID)
	if err != nil {
		return nil, "", err
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user)
	if err != nil {
		return nil, "", err
	}

	newRawRefresh, newRefreshHash, err := s.jwtService.GenerateOpaqueRefreshToken()
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()
	newRT := &model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: newRefreshHash,
		ExpiresAt: now.Add(s.refreshTTL),
		Revoked:   false,
		CreatedAt: now,
	}

	if err := s.authRepo.RotateRefreshToken(ctx, storedToken.ID, newRT); err != nil {
		return nil, "", err
	}

	return &model.AuthResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTTL.Seconds()),
		User:        user,
	}, newRawRefresh, nil
}
