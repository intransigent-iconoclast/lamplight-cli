package service

import (
	"context"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
)

type TorznabBackend struct {
	client *client.TorznabClient
}

func NewTorznabBackend(client *client.TorznabClient) *TorznabBackend {
	return &TorznabBackend{client: client}
}

func (b *TorznabBackend) Supports(indexerType entity.IndexerType) bool {
	return indexerType == entity.IndexerTypeTorznab
}

func (b *TorznabBackend) Search(ctx context.Context, indexer entity.Indexer, request dao.SearchRequest) ([]dao.SearchResult, error) {
	query := b.constructQueryFromRequest(&request)

	// b.client.fetch(ctx, query, indexer)
	xml, err := b.client.Fetch(ctx, query, indexer)
	if err != nil {
		return []dao.SearchResult{}, err
	}
	res, err := utils.ParseTorznabXml(xml)
	if err != nil {
		return []dao.SearchResult{}, err
	}
	for i := range res {
		res[i].IndexerName = indexer.Name
		res[i].Format = string(utils.DetectFormat(res[i].FormatAttr, res[i].Categories, res[i].Title))
	}
	return res, nil
}

func (b *TorznabBackend) constructQueryFromRequest(request *dao.SearchRequest) utils.TorznabQuery {
	query := utils.NewTorznabQuery()
	if request.Limit > 0 {
		limit := request.Limit
		query.Limit = &limit
	}
	if request.Query != "" {
		query.Query = request.Query
	}
	// worreid because idk if most indexers even support book lol
	// if query.Author != "" || query.Title != "" || query.Isbn != "" {
	// 	query.SearchType = utils.BOOK
	// }
	return query
}
