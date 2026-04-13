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

func (r *HistoryRepository) DeleteAll(ctx context.Context) error {
	return r.db.WithContext(ctx).Where("1 = 1").Delete(&entity.DownloadHistory{}).Error
}
