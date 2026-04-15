package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"gorm.io/gorm"
)

type ProviderRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) *ProviderRepository {
	return &ProviderRepository{
		db: db,
	}
}

func (r *ProviderRepository) FindAllProviders(ctx context.Context) ([]entity.Provider, error) {
	var providers []entity.Provider

	err := r.db.WithContext(ctx).Find(&providers).Error
	if err != nil {
		return nil, err
	}

	return providers, nil
}

func (r *ProviderRepository) SaveProvider(ctx context.Context, provider *entity.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}

	if err := r.db.WithContext(ctx).Create(provider).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("provider already exists")
		}
		return fmt.Errorf("create provider: %w", err)
	}
	return nil
}

func (r *ProviderRepository) DeleteByName(ctx context.Context, name string) error {
	result := r.db.WithContext(ctx).
		Where("name = ?", name).
		Delete(&entity.Provider{})
	if result.Error != nil {
		return fmt.Errorf("delete provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("no provider named %q found", name)
	}
	return nil
}

func (r *ProviderRepository) UpdateProvider(ctx context.Context, provider *entity.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}

	// SAVE With ID 0 == insert in gorm crazy bro don't ... save is an upsert command according to the docs exactly what I want here
	if provider.ID == 0 {
		return fmt.Errorf("provider ID is required for update")
	}

	tx := r.db.WithContext(ctx).Save(provider)
	if tx.Error != nil {
		return fmt.Errorf("update provider: %w", tx.Error)
	}

	if tx.RowsAffected == 0 {
		return fmt.Errorf("provider not found")
	}

	return nil
}
