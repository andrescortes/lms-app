package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear any existing env vars
	_ = os.Unsetenv("LMS_CORE_HOST")
	_ = os.Unsetenv("LMS_CORE_PORT")
	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("LMS_CORE_DB_PATH")

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host '127.0.0.1', got '%s'", cfg.Host)
	}
	if cfg.Port != "8082" {
		t.Errorf("expected Port '8082', got '%s'", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env 'development', got '%s'", cfg.Env)
	}
	if cfg.DBPath != "./data/lms-core.db" {
		t.Errorf("expected DBPath './data/lms-core.db', got '%s'", cfg.DBPath)
	}
}

func TestLoad_CustomValues(t *testing.T) {
	_ = os.Setenv("LMS_CORE_HOST", "0.0.0.0")
	_ = os.Setenv("LMS_CORE_PORT", "9090")
	_ = os.Setenv("APP_ENV", "production")
	_ = os.Setenv("LMS_CORE_DB_PATH", "/custom/path/db.sqlite")

	defer func() {
		_ = os.Unsetenv("LMS_CORE_HOST")
		_ = os.Unsetenv("LMS_CORE_PORT")
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("LMS_CORE_DB_PATH")
	}()

	cfg := Load()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host '0.0.0.0', got '%s'", cfg.Host)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got '%s'", cfg.Port)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env 'production', got '%s'", cfg.Env)
	}
	if cfg.DBPath != "/custom/path/db.sqlite" {
		t.Errorf("expected DBPath '/custom/path/db.sqlite', got '%s'", cfg.DBPath)
	}
}

func TestLoad_EmptyValuesUseDefaults(t *testing.T) {
	_ = os.Setenv("LMS_CORE_HOST", "")
	_ = os.Setenv("LMS_CORE_PORT", "")
	_ = os.Setenv("APP_ENV", "")
	_ = os.Setenv("LMS_CORE_DB_PATH", "")

	defer func() {
		_ = os.Unsetenv("LMS_CORE_HOST")
		_ = os.Unsetenv("LMS_CORE_PORT")
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("LMS_CORE_DB_PATH")
	}()

	cfg := Load()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected Host '127.0.0.1', got '%s'", cfg.Host)
	}
	if cfg.Port != "8082" {
		t.Errorf("expected Port '8082', got '%s'", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env 'development', got '%s'", cfg.Env)
	}
	if cfg.DBPath != "./data/lms-core.db" {
		t.Errorf("expected DBPath './data/lms-core.db', got '%s'", cfg.DBPath)
	}
}

func TestLoad_PartialEnv(t *testing.T) {
	_ = os.Setenv("LMS_CORE_HOST", "custom-host")
	// Don't set other vars

	defer func() {
		_ = os.Unsetenv("LMS_CORE_HOST")
	}()

	cfg := Load()

	if cfg.Host != "custom-host" {
		t.Errorf("expected Host 'custom-host', got '%s'", cfg.Host)
	}
	if cfg.Port != "8082" {
		t.Errorf("expected Port '8082' (default), got '%s'", cfg.Port)
	}
	if cfg.Env != "development" {
		t.Errorf("expected Env 'development' (default), got '%s'", cfg.Env)
	}
	if cfg.DBPath != "./data/lms-core.db" {
		t.Errorf("expected DBPath './data/lms-core.db' (default), got '%s'", cfg.DBPath)
	}
}