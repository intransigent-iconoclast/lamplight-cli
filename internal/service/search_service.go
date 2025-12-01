package service

import (
	"context"
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
)

// type definition
type SearchService struct {
	indexerRepository repository.IndexerRepository
	searchBackends    []SearchBackend
}

// constructor ... ugly but true/
// within () required params to construct type, and *SearchService is
// how structs are passed around in go is usually as pointers so that we arent' coppying structs around
// this is just how go creates structs :shrug:
func NewSearchService(
	indexerRepository repository.IndexerRepository,
	searchBackends []SearchBackend,
) *SearchService {
	return &SearchService{
		indexerRepository: indexerRepository,
		searchBackends:    searchBackends,
	}
}

func (s *SearchService) Search(ctx context.Context, request dao.SearchRequest) ([]dao.SearchResult, error) {
	targets, err := s.resolveIndexers(ctx, request)
	if err != nil {
		return nil, err
	}

	var all []dao.SearchResult
	for _, idx := range targets {
		backend, err := s.resolveBackend(idx.IndexerType)
		if err != nil {
			return nil, err
		}
		results, err := backend.Search(ctx, idx, request)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
	}

	// TODO: extra filtering on all based on req fields maybe later
	return all, nil
}

// perhaps revisit the design of this one right here ...
func (s *SearchService) resolveIndexers(ctx context.Context, request dao.SearchRequest) ([]entity.Indexer, error) {
	if request.IndexerName != "" {
		indexers, err := s.indexerRepository.FindByName(ctx, request.IndexerName)
		// was error
		if err != nil {
			return nil, err
		}
		// no indexers found return empty list
		if indexers == nil {
			return []entity.Indexer{}, nil
		}
		// found results
		return []entity.Indexer{*indexers}, nil
	}
	// if all else fails... just return indexers that are enabled
	return s.indexerRepository.FindByEnabled(ctx)
}

// picks A backend that supports the requested indexer type
func (s *SearchService) resolveBackend(indexerType entity.IndexerType) (SearchBackend, error) {
	for _, b := range s.searchBackends {
		if b.Supports(indexerType) {
			return b, nil
		}
	}
	return nil, fmt.Errorf("No backend for type: %s", indexerType)
}
