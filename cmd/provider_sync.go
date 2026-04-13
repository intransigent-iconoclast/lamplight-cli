package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var providerSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync indexers from configured providers",
	Long: `Fetch configured indexers from enabled providers (Jackett / Prowlarr)
and add them to the Lamplight indexer table.

By default only indexers that report book category support (7000 range)
are synced. Use --all to sync every indexer regardless of categories.
If an indexer does not report any capabilities it is included by default.

This command is idempotent:
- Existing indexers are skipped
- Disabled providers are ignored
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		providerRepo := repository.NewProviderRepository(db)
		indexerRepo := repository.NewIndexerRepository(db)

		providers, err := providerRepo.FindAllProviders(ctx)
		if err != nil {
			return fmt.Errorf("load providers: %w", err)
		}

		if len(providers) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No providers configured.")
			return nil
		}

		syncAll, _ := cmd.Flags().GetBool("all")

		added := 0
		skipped := 0

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
				return fmt.Errorf(
					"provider %q (%s): %w",
					provider.Name,
					provider.Type,
					err,
				)
			}

			for _, idx := range indexers {
				// Skip indexers that don't support books, unless --all is set.
				// If an indexer reports no caps at all, include it (fail open).
				if !syncAll && len(idx.Caps) > 0 && !indexerSupportsBooks(idx) {
					skipped++
					continue
				}

				name := fmt.Sprintf(
					"%s_%s",
					sanitize(idx.Name),
					sanitize(provider.Name),
				)

				var baseURL string
				switch provider.Type {
				case entity.ProviderTypeJackett:
					baseURL = fmt.Sprintf(
						"%s://%s:%d/api/v2.0/indexers/%s/results/torznab/",
						provider.Scheme,
						provider.Host,
						provider.Port,
						idx.ExternalID,
					)
				case entity.ProviderTypeProwlarr:
					baseURL = fmt.Sprintf(
						"%s://%s:%d/%s/api",
						provider.Scheme,
						provider.Host,
						provider.Port,
						idx.ExternalID,
					)
				}

				newIndexer := entity.Indexer{
					Name:        name,
					BaseURL:     baseURL,
					APIKey:      provider.APIKey,
					IndexerType: entity.IndexerTypeTorznab,
					Enabled:     true,
				}

				changed, err := indexerRepo.UpsertFromProvider(ctx, &newIndexer)
				if err != nil {
					return fmt.Errorf("save indexer %q: %w", name, err)
				}

				if changed {
					added++
				} else {
					skipped++
				}
			}
		}

		fmt.Fprintf(
			cmd.OutOrStdout(),
			"Sync complete: %d added, %d skipped\n",
			added,
			skipped,
		)

		return nil
	},
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

func sanitize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func init() {
	providerCmd.AddCommand(providerSyncCmd)
	providerSyncCmd.Flags().BoolP("all", "a", false, "Sync all indexers regardless of category support")
}
