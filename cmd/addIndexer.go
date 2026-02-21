/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
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
	Short: "This command adds an indexer manually that you will be able to query.",
	// this is where chatgpt shines :)
	Long: `Add a new indexer record to the Lamplight database.

An "indexer" is a search source (for example, a Jackett or Prowlarr
Torznab endpoint) that Lamplight can query for results.

You must provide at least:
  - a unique name (logical label, e.g. "The Pirate Bay")
  - the base URL for the indexer
Optionally, you can provide an API key and indexer type.

Examples:

  # Add a Torznab indexer pointing at a Jackett instance
  lamplight indexer add \
    --name "thepiratebay" \
    --base-url "http://127.0.0.1:9117/api/v2.0/indexers/thepiratebay/results/torznab/api" \
    --api-key "YOUR_JACKETT_API_KEY" \
    --indexer-type TORZNAB

  # Add another indexer with a different logical name
  lamplight indexer add \
    -n "rarbg" \
    -u "http://127.0.0.1:9117/api/v2.0/indexers/rarbg/results/torznab/api" \
    -k "YOUR_JACKETT_API_KEY" \
    -t TORZNAB

After adding an indexer, other commands can use it to run searches.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Validate this later
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
