// Package database owns the lifecycle of the PostgreSQL connection:
// opening it, configuring the connection pool, and closing it
// cleanly on shutdown. Nothing outside this package (and the
// repository layer, later) should know that GORM or "database/sql"
// is even involved.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Connect opens a connection to PostgreSQL using the given database
// config and returns a ready-to-use *gorm.DB.
//
// ctx is used to bound how long we're willing to wait for the
// initial "is the database actually reachable" ping. If Neon is
// down or the DSN is wrong, we fail fast instead of hanging forever.
func Connect(ctx context.Context, cfg config.DatabaseConfig) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		// TranslateError turns raw PostgreSQL errors (e.g. a unique
		// constraint violation) into GORM's portable error values
		// (e.g. gorm.ErrDuplicatedKey) so the service layer can check
		// `errors.Is(err, gorm.ErrDuplicatedKey)` instead of parsing
		// Postgres-specific error strings.
		TranslateError: true,
		Logger:         gormlogger.Default.LogMode(gormlogger.Warn),
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("database: opening connection: %w", err)
	}

	// gorm.DB wraps a *sql.DB, which is the actual connection pool.
	// We pull it out to configure pooling and to run a real ping.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: getting underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("database: pinging database: %w", err)
	}

	slog.Info("database connection established",
		"max_open_conns", cfg.MaxOpenConns,
		"max_idle_conns", cfg.MaxIdleConns,
	)

	return db, nil
}

// Close releases the underlying connection pool. Call this once,
// during graceful shutdown.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: getting underlying sql.DB: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("database: closing connection: %w", err)
	}
	return nil
}
