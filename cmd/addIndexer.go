package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// addIndexerCmd represents the addIndexer command
var addIndexerCmd = &cobra.Command{
	Use:   "add",
	Short: "add an indexer manually.",
	Long: `add a torznab indexer directly — useful if you're not using Prowlarr or Jackett
and just have a raw torznab endpoint you want to query.

most users should run 'lamplight provider sync' instead of this.

  lamplight indexer add \
    --name "thepiratebay" \
    --base-url "http://localhost:9117/api/v2.0/indexers/thepiratebay/results/torznab/api" \
    --api-key "YOUR_API_KEY"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			return fmt.Errorf("get flag 'name': %w", err)
		}

		baseURL, err := cmd.Flags().GetString("base-url")
		if err != nil {
			return fmt.Errorf("get flag 'base-url': %w", err)
		}

		apiKey, err := cmd.Flags().GetString("api-key")
		if err != nil {
			return fmt.Errorf("get flag 'api-key': %w", err)
		}

		rawType, err := cmd.Flags().GetString("indexer-type")
		if err != nil {
			return fmt.Errorf("get flag 'indexer-type': %w", err)
		}

		// normalize + map string → enum
		rawType = strings.ToUpper(strings.TrimSpace(rawType))

		var idxType entity.IndexerType
		switch rawType {
		case string(entity.IndexerTypeTorznab):
			idxType = entity.IndexerTypeTorznab
		default:
			return fmt.Errorf("unsupported indexer type %q; valid types: %s",
				rawType, string(entity.IndexerTypeTorznab))
		}

		// use command context so it's cancellable if needed
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewIndexerRepository(db)

		// construct the thing to save ..
		idx := entity.Indexer{
			Name:        name,
			BaseURL:     baseURL,
			APIKey:      apiKey,
			IndexerType: idxType,
			Enabled:     true,
		}

		if err := repo.SaveIndexer(ctx, &idx); err != nil {
			return fmt.Errorf("save indexer: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added indexer %q (id=%d)\n", idx.Name, idx.ID)
		return nil

	},
}

func init() {
	indexerCmd.AddCommand(addIndexerCmd)

	addIndexerCmd.Flags().StringP("name", "n", "", "name of indexer i.e., The Pirate Bay")
	addIndexerCmd.Flags().StringP("base-url", "u", "", "base jackett url for indexer copy from jackett.")
	addIndexerCmd.Flags().StringP("api-key", "k", "", "The api key from jackett. Get from Jackett")
	addIndexerCmd.Flags().StringP("indexer-type", "t", string(entity.IndexerTypeTorznab), "The type of indexer. Select from: TORZNAB, ")

	addIndexerCmd.MarkFlagRequired("name")
	addIndexerCmd.MarkFlagRequired("base-url")
}
