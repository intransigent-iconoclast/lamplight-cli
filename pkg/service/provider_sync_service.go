package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
)

// ProviderSyncService pulls indexers from configured providers (Prowlarr/Jackett)
// into the indexer table. It is idempotent: existing indexers are skipped and
// disabled providers are ignored.
type ProviderSyncService struct {
	providerRepo *repository.ProviderRepository
	indexerRepo  *repository.IndexerRepository
}

func NewProviderSyncService(providerRepo *repository.ProviderRepository, indexerRepo *repository.IndexerRepository) *ProviderSyncService {
	return &ProviderSyncService{providerRepo: providerRepo, indexerRepo: indexerRepo}
}

// SyncReport summarizes the outcome of a sync pass.
type SyncReport struct {
	Added   int
	Skipped int
}

// Sync fetches indexers from every enabled provider and upserts the ones that
// support books (or all of them, when all is true).
func (s *ProviderSyncService) Sync(ctx context.Context, all bool) (*SyncReport, error) {
	providers, err := s.providerRepo.FindAllProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("load providers: %w", err)
	}

	report := &SyncReport{}

	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}

		var providerClient client.ProviderClient
		switch provider.Type {
		case entity.ProviderTypeJackett:
			providerClient = client.NewJackettClient()
		case entity.ProviderTypeProwlarr:
			providerClient = client.NewProwlarrClient()
		default:
			continue
		}

		indexers, err := providerClient.RetrieveIndexers(ctx, &provider)
		if err != nil {
			return report, fmt.Errorf("provider %q (%s): %w", provider.Name, provider.Type, err)
		}

		for _, idx := range indexers {
			// Skip indexers that don't support books, unless `all` is set.
			// If an indexer reports no caps at all, include it (fail open).
			if !all && len(idx.Caps) > 0 && !indexerSupportsBooks(idx) {
				report.Skipped++
				continue
			}

			name := fmt.Sprintf("%s_%s", sanitizeName(idx.Name), sanitizeName(provider.Name))

			var baseURL string
			switch provider.Type {
			case entity.ProviderTypeJackett:
				baseURL = fmt.Sprintf("%s://%s:%d/api/v2.0/indexers/%s/results/torznab/",
					provider.Scheme, provider.Host, provider.Port, idx.ExternalID)
			case entity.ProviderTypeProwlarr:
				baseURL = fmt.Sprintf("%s://%s:%d/%s/api",
					provider.Scheme, provider.Host, provider.Port, idx.ExternalID)
			}

			newIndexer := entity.Indexer{
				Name:        name,
				BaseURL:     baseURL,
				APIKey:      provider.APIKey,
				IndexerType: entity.IndexerTypeTorznab,
				Enabled:     true,
			}

			changed, err := s.indexerRepo.UpsertFromProvider(ctx, &newIndexer)
			if err != nil {
				return report, fmt.Errorf("save indexer %q: %w", name, err)
			}
			if changed {
				report.Added++
			} else {
				report.Skipped++
			}
		}
	}

	return report, nil
}

func indexerSupportsBooks(idx dao.ProviderIndexerDAO) bool {
	allowed := make(map[int]struct{}, len(constants.BookCategories))
	for _, c := range constants.BookCategories {
		allowed[c] = struct{}{}
	}
	for _, c := range idx.Caps {
		if _, ok := allowed[c]; ok {
			return true
		}
	}
	return false
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
