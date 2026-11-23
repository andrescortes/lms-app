package config

import (
	"os"
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
	LMSCoreURL      string
}

func Load() *Config {
	env := getEnv("APP_ENV", "development")
	return &Config{
		Port:            getEnv("GATEWAY_PORT", "8081"),
		Env:             env,
		DBPath:          getEnv("GATEWAY_DB_PATH", "./data/gateway-auth.db"),
		JWTSecret:       getEnv("JWT_SECRET", "dev-super-secret-key-change-in-production-min-32-chars"),
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		CookieSecure:    env == "production",
		LMSCoreURL:      getEnv("LMS_CORE_URL", "http://127.0.0.1:8082"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
