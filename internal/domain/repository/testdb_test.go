package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/KhalidMohomud/ecomApi/internal/config"
	"github.com/KhalidMohomud/ecomApi/internal/database"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// testDB opens a real connection to the database configured in
// .env, using the exact same config/database packages production
// code uses, and registers cleanup to close it when the test
// finishes. Every repository's integration test (user, category,
// brand, and whatever catalog entities follow) shares this one
// connection helper instead of each re-implementing it.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg, err := config.Load(repoRootEnvFile(t))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, cfg.Database)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, database.Close(db))
	})

	return db
}
