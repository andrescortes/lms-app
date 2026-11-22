package config

import "os"

type Config struct {
	Port   string
	Env    string
	DBPath string
}

func Load() *Config {
	return &Config{
		Port:   getEnv("GATEWAY_PORT", "8081"),
		Env:    getEnv("APP_ENV", "development"),
		DBPath: getEnv("GATEWAY_DB_PATH", "./data/gateway-auth.db"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
