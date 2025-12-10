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
	indexerRepository *repository.IndexerRepository
	searchBackends    []SearchBackend
}

// constructor ... ugly but true.
// within () required params to construct type, and *SearchService is
// how structs are passed around in go is usually as pointers so that we arent' coppying structs around
// this is just how go creates structs :shrug:
func NewSearchService(
	indexerRepository *repository.IndexerRepository,
	searchBackends []SearchBackend,
) *SearchService {
	return &SearchService{
		indexerRepository: indexerRepository,
		searchBackends:    searchBackends,
	}
}

func (s *SearchService) Search(ctx context.Context, request dao.SearchRequest, criteria dao.FilterCriteria) ([]dao.SearchResult, error) {
	// resolve which indexers to hit (by name or all enabled)
	indexers, err := s.getIndexers(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(indexers) == 0 {
		return []dao.SearchResult{}, nil
	}

	var all []dao.SearchResult

	// for each indexer, pick a backend and call it!
	var lastErr error
	for _, idx := range indexers {
		backend, err := s.resolveBackend(idx.IndexerType)
		// busted af just skip this one ... probably log this so user knows I'm never gonna use this ... not even sure how this would happen but
		if err != nil {
			lastErr = fmt.Errorf("no backend for indexer %q (type=%s): %w", idx.Name, idx.IndexerType, err)
			continue
		}

		results, err := backend.Search(ctx, idx, request)
		if err != nil {
			// no results found skiiiip
			lastErr = fmt.Errorf("search with indexer %q (id=%d): %w", idx.Name, idx.ID, err)
			continue
		}

		all = append(all, results...)
	}

	// everything failed not sure if this is a good idea or not but :shrug
	if len(all) == 0 && lastErr != nil {
		return nil, lastErr
	}

	// filtering!!!
	filteredHits := FilterResults(all, criteria)

	// some results came back
	return filteredHits, nil
}

// picks A backend that supports the requested indexer type
func (s *SearchService) resolveBackend(indexerType entity.IndexerType) (SearchBackend, error) {
	for _, b := range s.searchBackends {
		if b.Supports(indexerType) {
			return b, nil
		}
	}
	return nil, fmt.Errorf("no backend for type: %s", indexerType)
}

func (s *SearchService) getIndexers(ctx context.Context, request dao.SearchRequest) ([]entity.Indexer, error) {
	// Specific indexer requested by name: return at most one.

	if request.IndexerName != "" {
		indexer, err := s.indexerRepository.FindByName(ctx, request.IndexerName)
		if err != nil {
			return nil, err
		}
		if indexer == nil {
			// No such indexer; treat as "no targets".
			return []entity.Indexer{}, nil
		}
		return []entity.Indexer{*indexer}, nil
	}

	// No indexer specified: use all enabled.
	indexers, err := s.indexerRepository.FindByEnabled(ctx)
	if err != nil {
		return nil, err
	}
	return indexers, nil
}
