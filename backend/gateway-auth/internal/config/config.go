package config

import "os"

type Config struct {
	Port string
	Env  string
}

func Load() *Config {
	return &Config{
		Port: getEnv("GATEWAY_PORT", "8081"),
		Env:  getEnv("APP_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
