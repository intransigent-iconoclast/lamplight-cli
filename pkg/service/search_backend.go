package service

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
)

type SearchBackend interface {
	Supports(indexerType entity.IndexerType) bool
	Search(ctx context.Context, indexer entity.Indexer, request dao.SearchRequest) ([]dao.SearchResult, error)
}
