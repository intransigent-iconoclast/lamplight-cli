package repository

import (
	"context"
	"errors"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type LibraryConfigRepository struct {
	db *gorm.DB
}

func NewLibraryConfigRepository(db *gorm.DB) *LibraryConfigRepository {
	return &LibraryConfigRepository{db: db}
}

// Get returns the config, creating a default one if it doesn't exist yet.
func (r *LibraryConfigRepository) Get(ctx context.Context) (*entity.LibraryConfig, error) {
	var cfg entity.LibraryConfig
	err := r.db.WithContext(ctx).Session(&gorm.Session{Logger: r.db.Logger.LogMode(logger.Silent)}).First(&cfg, 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cfg = entity.LibraryConfig{
			ID:          1,
			LibraryPath: entity.DefaultLibraryPath,
			Template:    entity.DefaultTemplate,
		}
		if createErr := r.db.WithContext(ctx).Create(&cfg).Error; createErr != nil {
			return nil, createErr
		}
		return &cfg, nil
	}
	return &cfg, err
}

// Save upserts the config row (always ID=1).
func (r *LibraryConfigRepository) Save(ctx context.Context, cfg *entity.LibraryConfig) error {
	cfg.ID = 1
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"library_path", "template", "deluge_path", "host_path"}),
	}).Create(cfg).Error
}
