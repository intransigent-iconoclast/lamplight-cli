package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"gorm.io/gorm"
)

// type definition
type IndexerRepository struct {
	db *gorm.DB
}

// constructor
func NewIndexerRepository(db *gorm.DB) *IndexerRepository {
	return &IndexerRepository{db: db}
}

func (r *IndexerRepository) FindByName(ctx context.Context, name string) (*entity.Indexer, error) {
	var indexer entity.Indexer
	err := r.db.WithContext(ctx).
		Where("name = ?", name).
		Take(&indexer).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// result not found no error
			return nil, nil
		}
		// error encoutered
		return nil, err
	}
	// indexer found
	return &indexer, nil
}

func (r *IndexerRepository) FindByEnabled(ctx context.Context) ([]entity.Indexer, error) {
	var indexers []entity.Indexer
	err := r.db.WithContext(ctx).
		Where("enabled = ?", true).
		Find(&indexers).
		Error

	if err != nil {
		return nil, err
	}
	// fmt.Printf("%+v\n", indexers)
	return indexers, nil
}

func (r *IndexerRepository) FindAllIndexers(ctx context.Context) ([]entity.Indexer, error) {
	var indexers []entity.Indexer
	// hate that the lint doesn't like the '.' on the new line
	err := r.db.WithContext(ctx).Order("id ASC").
		Find(&indexers).Error

	if err != nil {
		return nil, err
	}
	return indexers, nil
}

func (r *IndexerRepository) SaveIndexer(ctx context.Context, indexer *entity.Indexer) error {
	if indexer == nil {
		return fmt.Errorf("indexer is nil")
	}

	// this will return an error if the indexer already exists
	if err := r.db.WithContext(ctx).Create(indexer).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return utils.ErrIndexerExists
		}
		return fmt.Errorf("create indexer: %w", err)
	}
	return nil
}

func (r *IndexerRepository) DeleteIndexerById(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&entity.Indexer{}, id)
	if result.Error != nil {
		return result.Error
	}
	// if no rows were modified this item should be emitted ... might refactor cuz
	if result.RowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil

}

// func (r *IndexerRepository) DumpAll(ctx context.Context) error {
// 	var idxs []entity.Indexer
// 	err := r.db.WithContext(ctx).Debug().Find(&idxs).Error
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Printf("ALL INDEXERS FROM GO: %#v\n", idxs)
// 	return nil
// }
