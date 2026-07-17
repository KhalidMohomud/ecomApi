// Package config is responsible for loading and validating all
// application configuration from environment variables.
//
// No other package in this project is allowed to call os.Getenv
// directly. Every setting the app needs flows through the Config
// struct defined here, so there is exactly one place that knows
// where configuration comes from.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds every configuration value the application needs,
// grouped by the subsystem that owns them.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Auth     AuthConfig
}

// AppConfig holds general application/server settings.
type AppConfig struct {
	Env  string // "development", "staging", "production"
	Port string
}

// DatabaseConfig holds everything needed to open a PostgreSQL connection.
type DatabaseConfig struct {
	DSN             string        // full connection string, e.g. Neon's postgres:// URL
	MaxOpenConns    int           // max simultaneous open connections to the DB
	MaxIdleConns    int           // max connections kept idle in the pool
	ConnMaxLifetime time.Duration // max time a connection may be reused
}

// AuthConfig holds everything needed to issue and validate tokens.
type AuthConfig struct {
	JWTSecret       string        // signing key for access tokens (HMAC)
	AccessTokenTTL  time.Duration // how long an access token stays valid
	RefreshTokenTTL time.Duration // how long a refresh token stays valid
}

// Load reads configuration from environment variables (optionally
// populated from a .env file) and returns a validated Config.
//
// envFile is the path to a .env file to load, e.g. ".env". Passing ""
// skips file loading outright and relies purely on real environment
// variables. Passing a real path (the normal case — main.go always
// calls Load(".env")) is safe even when that file doesn't exist: a
// missing file is expected outside local development, since a
// container or a real deployment sets environment variables directly
// instead of shipping a file, and .env is deliberately excluded from
// what gets built into the Docker image (see .dockerignore) so a
// secret never ends up baked into an image layer. Only a file that
// exists but fails to parse is treated as fatal — that indicates a
// real local misconfiguration worth stopping for.
func Load(envFile string) (*Config, error) {
	// godotenv.Load reads KEY=VALUE lines from the given file and
	// injects them into the process environment (os.Setenv), as if
	// you had exported them in your shell before running the app.
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("config: loading env file %q: %w", envFile, err)
		}
	}

	v := viper.New()

	// AutomaticEnv tells Viper to read values from the process
	// environment (the same place godotenv just wrote to) instead
	// of requiring a config file (config.yaml, config.json, ...).
	// This is what lets us call v.GetString("APP_PORT") below and
	// have it resolve to the APP_PORT environment variable.
	v.AutomaticEnv()

	// Defaults are used only when the environment variable is unset.
	// They keep local development working with a minimal .env file.
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("DB_MAX_OPEN_CONNS", 25)
	v.SetDefault("DB_MAX_IDLE_CONNS", 5)
	v.SetDefault("DB_CONN_MAX_LIFETIME_MINUTES", 5)
	v.SetDefault("JWT_ACCESS_TOKEN_TTL_MINUTES", 15)
	v.SetDefault("JWT_REFRESH_TOKEN_TTL_DAYS", 30)

	cfg := &Config{
		App: AppConfig{
			Env:  v.GetString("APP_ENV"),
			Port: v.GetString("APP_PORT"),
		},
		Database: DatabaseConfig{
			DSN:             v.GetString("DATABASE_URL"),
			MaxOpenConns:    v.GetInt("DB_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("DB_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(v.GetInt("DB_CONN_MAX_LIFETIME_MINUTES")) * time.Minute,
		},
		Auth: AuthConfig{
			JWTSecret:       v.GetString("JWT_SECRET"),
			AccessTokenTTL:  time.Duration(v.GetInt("JWT_ACCESS_TOKEN_TTL_MINUTES")) * time.Minute,
			RefreshTokenTTL: time.Duration(v.GetInt("JWT_REFRESH_TOKEN_TTL_DAYS")) * 24 * time.Hour,
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate fails fast if a required setting is missing. Failing at
// startup (instead of the first time the DB is queried) makes
// misconfiguration obvious immediately, rather than as a confusing
// runtime error minutes into a request.
func (c *Config) validate() error {
	if c.Database.DSN == "" {
		return fmt.Errorf("DATABASE_URL is required but was empty")
	}
	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET is required and must be at least 32 characters (got %d)", len(c.Auth.JWTSecret))
	}
	return nil
}

// IsProduction reports whether the app is running in production.
// Handlers/middleware use this to decide things like whether to
// return raw error messages to the client.
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
