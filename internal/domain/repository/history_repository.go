package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"gorm.io/gorm"
)

type HistoryRepository struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

func (r *HistoryRepository) Save(ctx context.Context, entry *entity.DownloadHistory) error {
	if entry == nil {
		return fmt.Errorf("history entry is nil")
	}
	entry.DownloadedAt = time.Now()
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *HistoryRepository) FindAll(ctx context.Context) ([]entity.DownloadHistory, error) {
	var entries []entity.DownloadHistory
	err := r.db.WithContext(ctx).Order("downloaded_at DESC").Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *HistoryRepository) ExistsByLink(ctx context.Context, link string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&entity.DownloadHistory{}).
		Where("link = ?", link).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *HistoryRepository) FindByID(ctx context.Context, id uint) (*entity.DownloadHistory, error) {
	var entry entity.DownloadHistory
	err := r.db.WithContext(ctx).First(&entry, id).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (r *HistoryRepository) UpdateStatus(ctx context.Context, id uint, status entity.DownloadStatus) error {
	result := r.db.WithContext(ctx).
		Model(&entity.DownloadHistory{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no entry found with id %d", id)
	}
	return nil
}

// FindActive returns all entries that are still in-flight (snatched or downloading)
// and have a torrent hash we can poll on.
func (r *HistoryRepository) FindActive(ctx context.Context) ([]entity.DownloadHistory, error) {
	var entries []entity.DownloadHistory
	err := r.db.WithContext(ctx).
		Where("status IN ? AND torrent_hash != ''", []string{"snatched", "downloading"}).
		Order("downloaded_at DESC").
		Find(&entries).Error
	return entries, err
}

// FindCompleted returns entries that finished downloading but haven't been organized yet.
func (r *HistoryRepository) FindCompleted(ctx context.Context) ([]entity.DownloadHistory, error) {
	var entries []entity.DownloadHistory
	err := r.db.WithContext(ctx).
		Where("status = ? AND file_path != ''", entity.StatusCompleted).
		Order("downloaded_at DESC").
		Find(&entries).Error
	return entries, err
}

func (r *HistoryRepository) UpdateStatusAndPath(ctx context.Context, id uint, status entity.DownloadStatus, filePath string) error {
	result := r.db.WithContext(ctx).
		Model(&entity.DownloadHistory{}).
		Where("id = ?", id).
		Updates(map[string]any{"status": status, "file_path": filePath})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no entry found with id %d", id)
	}
	return nil
}

func (r *HistoryRepository) DeleteAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("1 = 1").Delete(&entity.DownloadHistory{}).Error
}
