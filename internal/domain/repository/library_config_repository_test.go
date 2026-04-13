package repository

import (
	"context"
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openConfigTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.LibraryConfig{}))
	return db
}

func TestLibraryConfigRepo_Get_CreatesDefaultOnFirstCall(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	cfg, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint(1), cfg.ID)
	assert.Equal(t, entity.DefaultLibraryPath, cfg.LibraryPath)
	assert.Equal(t, entity.DefaultTemplate, cfg.Template)
}

func TestLibraryConfigRepo_Get_CalledTwice_DoesNotDuplicate(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	_, err := repo.Get(ctx)
	require.NoError(t, err)
	_, err = repo.Get(ctx)
	require.NoError(t, err)

	// should still only be one row
	var count int64
	db := openConfigTestDB(t)
	db.Model(&entity.LibraryConfig{}).Count(&count)
	assert.LessOrEqual(t, count, int64(1))
}

func TestLibraryConfigRepo_Save_UpdatesLibraryPath(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	// create default first
	_, err := repo.Get(ctx)
	require.NoError(t, err)

	err = repo.Save(ctx, &entity.LibraryConfig{
		LibraryPath: "/mnt/media/books",
		Template:    entity.DefaultTemplate,
	})
	require.NoError(t, err)

	cfg, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/mnt/media/books", cfg.LibraryPath)
}

func TestLibraryConfigRepo_Save_UpdatesTemplate(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	_, _ = repo.Get(ctx) // init default

	require.NoError(t, repo.Save(ctx, &entity.LibraryConfig{
		LibraryPath: entity.DefaultLibraryPath,
		Template:    "{author}/{title}",
	}))

	cfg, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "{author}/{title}", cfg.Template)
}

func TestLibraryConfigRepo_Save_UpdatesPathMapping(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	_, _ = repo.Get(ctx)

	require.NoError(t, repo.Save(ctx, &entity.LibraryConfig{
		LibraryPath: entity.DefaultLibraryPath,
		Template:    entity.DefaultTemplate,
		DelugePath:  "/data/incomplete",
		HostPath:    "/mnt/media/incomplete",
	}))

	cfg, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/data/incomplete", cfg.DelugePath)
	assert.Equal(t, "/mnt/media/incomplete", cfg.HostPath)
}

func TestLibraryConfigRepo_Save_AlwaysForcesIDTo1(t *testing.T) {
	repo := NewLibraryConfigRepository(openConfigTestDB(t))
	ctx := context.Background()

	// attempt to save with a different ID — should be coerced to 1
	err := repo.Save(ctx, &entity.LibraryConfig{
		ID:          99,
		LibraryPath: "/custom/path",
		Template:    entity.DefaultTemplate,
	})
	require.NoError(t, err)

	cfg, err := repo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint(1), cfg.ID)
	assert.Equal(t, "/custom/path", cfg.LibraryPath)
}
