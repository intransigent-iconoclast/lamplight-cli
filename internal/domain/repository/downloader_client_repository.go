package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"gorm.io/gorm"
)

type DownloaderRepository struct {
	db *gorm.DB
}

func NewDownloaderRepository(db *gorm.DB) *DownloaderRepository {
	return &DownloaderRepository{db: db}
}

func (r *DownloaderRepository) SaveClient(ctx context.Context, client *entity.Downloader) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	if strings.TrimSpace(client.Name) == "" {
		return fmt.Errorf("client name cannot be empty")
	}

	if err := r.db.WithContext(ctx).Create(client).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return utils.ErrClientExists
		}
		return fmt.Errorf("create client: %w", err)
	}

	return nil
}

func (r *DownloaderRepository) FindByDownloaderType(ctx context.Context, t entity.DownloaderType) (*entity.Downloader, error) {
	var dlClient entity.Downloader
	kind := string(t)
	err := r.db.WithContext(ctx).
		Where("client_type = ?", kind).
		Take(&dlClient).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &dlClient, nil
}

func (r *DownloaderRepository) FindAllDownloaders(ctx context.Context) ([]entity.Downloader, error) {
	var clients []entity.Downloader
	err := r.db.WithContext(ctx).Order("id ASC").Find(&clients).Error

	if err != nil {
		return nil, err
	}

	return clients, nil
}

// This function returns the highest priority download client.
func (r *DownloaderRepository) FindHighestPriorityDownloader(ctx context.Context) (*entity.Downloader, error) {
	var client entity.Downloader
	err := r.db.WithContext(ctx).Order("priority ASC").Limit(1).Take(&client).Error
	if err != nil {
		return nil, fmt.Errorf("error retrieving highest priority download client: %w", err)
	}
	return &client, nil
}

func (r *DownloaderRepository) DeleteByName(ctx context.Context, name string) error {
	result := r.db.WithContext(ctx).
		Where("name = ?", name).
		Delete(&entity.Downloader{})

	if result.Error != nil {
		return fmt.Errorf("delete client: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no downloader client named %q found", name)
	}

	return nil
}

func (r *DownloaderRepository) FindByName(ctx context.Context, name string) (*entity.Downloader, error) {
	var client entity.Downloader

	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		Take(&client).
		Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &client, nil
}

func (r *DownloaderRepository) Update(ctx context.Context, client *entity.Downloader) error {
	if client == nil {
		return fmt.Errorf("client is nil")
	}

	return r.db.WithContext(ctx).Save(client).Error
}
