package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"gorm.io/gorm"
)

type CacheRepository struct {
	db *gorm.DB
}

func NewCacheRepository(db *gorm.DB) *CacheRepository {
	return &CacheRepository{
		db: db,
	}
}

// This function adds the user's last search so that they can select a specific result to download
// if desired at a later date.
func (c *CacheRepository) AddResultsToCache(ctx context.Context, results *[]dao.SearchResult) error {
	j, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("Error Marshaling Results: %w", err)
	}

	row := entity.SearchCache{
		ID:     1,
		Result: string(j),
	}
	return c.db.WithContext(ctx).Save(&row).Error
}

// Get cached json array of values. Returns a string that MUST be UnMarshaled.
func (c *CacheRepository) GetResultsCache(ctx context.Context) (string, error) {
	var cache entity.SearchCache
	if err := c.db.First(&cache, 1).Error; err != nil {
		return "", err
	}
	return cache.Result, nil
}

// Returns full cache row including metadata (UpdatedAt)
func (c *CacheRepository) GetCache(ctx context.Context) (*entity.SearchCache, error) {
	var cache entity.SearchCache
	if err := c.db.First(&cache, 1).Error; err != nil {
		return nil, err
	}
	return &cache, nil
}
