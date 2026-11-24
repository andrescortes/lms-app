package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port            string
	Env             string
	DBPath          string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CookieSecure    bool
	CookieDomain    string
	LMSCoreURL      string
}

func Load() *Config {
	env := getEnv("APP_ENV", "development")
	jwtSecret := getEnv("JWT_SECRET", "dev-super-secret-key-change-in-production-min-32-chars")

	// Validate JWT secret in production
	if env == "production" && len(jwtSecret) < 32 {
		panic("JWT_SECRET must be at least 32 characters in production")
	}

	return &Config{
		Port:            getEnv("GATEWAY_PORT", "8081"),
		Env:             env,
		DBPath:          getEnv("GATEWAY_DB_PATH", "./data/gateway-auth.db"),
		JWTSecret:       jwtSecret,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieSecure:    env == "production",
		CookieDomain:    getEnv("COOKIE_DOMAIN", ""),
		LMSCoreURL:      getEnv("LMS_CORE_URL", "http://127.0.0.1:8082"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// ValidateLMSCoreURL ensures the LMS Core URL uses HTTPS in production
func (c *Config) ValidateLMSCoreURL() error {
	if c.Env == "production" && !strings.HasPrefix(c.LMSCoreURL, "https://") {
		return fmt.Errorf("LMS_CORE_URL must use HTTPS in production")
	}
	return nil
}
