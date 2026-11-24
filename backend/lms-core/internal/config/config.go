package config

import "os"

type Config struct {
	Host   string
	Port   string
	Env    string
	DBPath string
}

func Load() *Config {
	return &Config{
		Host:   getEnv("LMS_CORE_HOST", "127.0.0.1"),
		Port:   getEnv("LMS_CORE_PORT", "8082"),
		Env:    getEnv("APP_ENV", "development"),
		DBPath: getEnv("LMS_CORE_DB_PATH", "./data/lms-core.db"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}
