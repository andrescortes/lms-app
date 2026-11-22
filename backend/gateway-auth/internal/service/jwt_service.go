package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"lms-app/backend/gateway-auth/internal/model"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type CustomClaims struct {
	Email     string `json:"email"`
	Role      string `json:"role"`
	CEFRLevel string `json:"cefr_level"`
	jwt.RegisteredClaims
}

type JWTService struct {
	secretKey []byte
	accessTTL time.Duration
	issuer    string
}

func NewJWTService(secret string, accessTTL time.Duration) *JWTService {
	return &JWTService{secretKey: []byte(secret), accessTTL: accessTTL, issuer: "gateway-auth"}
}

func (j *JWTService) GenerateAccessToken(user *model.User) (string, error) {
	now := time.Now().UTC()

	claims := CustomClaims{
		Email:     user.Email,
		Role:      user.Role,
		CEFRLevel: user.CEFRLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTTL)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(j.secretKey)
	if err != nil {
		return "", fmt.Errorf("signing jwt access token: %w", err)
	}
	return signed, nil
}

func (j *JWTService) ValidateAccessToken(tokenString string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) {
			return j.secretKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(j.issuer),
	)

	if err != nil {
		return nil, fmt.Errorf("invalid access token: %w", err)
	}
	if !token.Valid {
		return nil, errors.New("token is not valid")
	}

	return claims, nil
}

func (j *JWTService) GenerateOpaqueRefreshToken() (rawToken string, tokenHash string, err error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("generating random refresh token: %w", err)
	}
	rawToken = base64.RawURLEncoding.EncodeToString(randomBytes)
	tokenHash = HashRefreshToken(rawToken)
	return rawToken, tokenHash, nil
}

func HashRefreshToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
