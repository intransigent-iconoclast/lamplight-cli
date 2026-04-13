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

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&entity.DownloadHistory{}))
	return db
}

func TestHistoryRepo_Save_And_FindAll(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{
		Title:  "Dune",
		Link:   "magnet:?xt=urn:btih:abc",
		Status: entity.StatusSnatched,
	}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{
		Title:  "Foundation",
		Link:   "magnet:?xt=urn:btih:def",
		Status: entity.StatusCompleted,
	}))

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestHistoryRepo_FindAll_OrderedByDownloadedAtDesc(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "First", Link: "a", Status: entity.StatusSnatched}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Second", Link: "b", Status: entity.StatusSnatched}))

	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	// most recent first
	assert.Equal(t, "Second", all[0].Title)
	assert.Equal(t, "First", all[1].Title)
}

func TestHistoryRepo_ExistsByLink_True(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	link := "magnet:?xt=urn:btih:abc123"
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Dune", Link: link, Status: entity.StatusSnatched}))

	exists, err := repo.ExistsByLink(ctx, link)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestHistoryRepo_ExistsByLink_False(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	exists, err := repo.ExistsByLink(ctx, "magnet:?xt=urn:btih:nothere")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestHistoryRepo_ExistsByLink_PartialMatch_ReturnsFalse(t *testing.T) {
	// ExistsByLink must be an exact match, not a substring match
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Dune", Link: "magnet:?xt=urn:btih:abc123", Status: entity.StatusSnatched}))

	exists, err := repo.ExistsByLink(ctx, "magnet:?xt=urn:btih:abc")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestHistoryRepo_FindByStatus(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "A", Link: "a", Status: entity.StatusFailed}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "B", Link: "b", Status: entity.StatusFailed}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "C", Link: "c", Status: entity.StatusCompleted}))

	failed, err := repo.FindByStatus(ctx, entity.StatusFailed)
	require.NoError(t, err)
	assert.Len(t, failed, 2)
	for _, e := range failed {
		assert.Equal(t, entity.StatusFailed, e.Status)
	}
}

func TestHistoryRepo_FindActive_OnlySnatchedAndDownloading(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Snatched", Link: "a", Status: entity.StatusSnatched, TorrentHash: "hash1"}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Downloading", Link: "b", Status: entity.StatusDownloading, TorrentHash: "hash2"}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Completed", Link: "c", Status: entity.StatusCompleted, TorrentHash: "hash3"}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "Failed", Link: "d", Status: entity.StatusFailed, TorrentHash: "hash4"}))

	active, err := repo.FindActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 2)
	for _, e := range active {
		assert.True(t, e.Status == entity.StatusSnatched || e.Status == entity.StatusDownloading)
	}
}

func TestHistoryRepo_FindActive_ExcludesEmptyHash(t *testing.T) {
	// snatched with no hash (e.g. hash wasn't saved properly) should NOT appear as active
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "NoHash", Link: "a", Status: entity.StatusSnatched, TorrentHash: ""}))

	active, err := repo.FindActive(ctx)
	require.NoError(t, err)
	assert.Len(t, active, 0)
}

func TestHistoryRepo_FindCompleted_OnlyWithFilePath(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "WithPath", Link: "a", Status: entity.StatusCompleted, FilePath: "/some/path/book.epub"}))
	require.NoError(t, repo.Save(ctx, &entity.DownloadHistory{Title: "NoPath", Link: "b", Status: entity.StatusCompleted, FilePath: ""}))

	completed, err := repo.FindCompleted(ctx)
	require.NoError(t, err)
	assert.Len(t, completed, 1)
	assert.Equal(t, "WithPath", completed[0].Title)
}

func TestHistoryRepo_UpdateStatus(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	entry := &entity.DownloadHistory{Title: "Dune", Link: "a", Status: entity.StatusSnatched}
	require.NoError(t, repo.Save(ctx, entry))

	require.NoError(t, repo.UpdateStatus(ctx, entry.ID, entity.StatusFailed))

	all, _ := repo.FindAll(ctx)
	assert.Equal(t, entity.StatusFailed, all[0].Status)
}

func TestHistoryRepo_UpdateStatus_NonExistentID_ReturnsError(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	err := repo.UpdateStatus(ctx, 9999, entity.StatusFailed)
	assert.Error(t, err)
}

func TestHistoryRepo_UpdateStatusAndHash(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	entry := &entity.DownloadHistory{Title: "Dune", Link: "a", Status: entity.StatusSnatched, TorrentHash: "oldhash"}
	require.NoError(t, repo.Save(ctx, entry))

	require.NoError(t, repo.UpdateStatusAndHash(ctx, entry.ID, entity.StatusDownloading, "newhash"))

	all, _ := repo.FindAll(ctx)
	assert.Equal(t, entity.StatusDownloading, all[0].Status)
	assert.Equal(t, "newhash", all[0].TorrentHash)
}

func TestHistoryRepo_UpdateStatusAndPath(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	ctx := context.Background()

	entry := &entity.DownloadHistory{Title: "Dune", Link: "a", Status: entity.StatusDownloading}
	require.NoError(t, repo.Save(ctx, entry))

	require.NoError(t, repo.UpdateStatusAndPath(ctx, entry.ID, entity.StatusCompleted, "/lib/Dune.epub"))

	all, _ := repo.FindAll(ctx)
	assert.Equal(t, entity.StatusCompleted, all[0].Status)
	assert.Equal(t, "/lib/Dune.epub", all[0].FilePath)
}

func TestHistoryRepo_Save_NilEntry_ReturnsError(t *testing.T) {
	repo := NewHistoryRepository(openTestDB(t))
	err := repo.Save(context.Background(), nil)
	assert.Error(t, err)
}
