/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
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
	Run: func(cmd *cobra.Command, args []string) {
		// validate inputs before spinning up any db
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("get flag 'name': %v", err)
		}

		baseUrl, err := cmd.Flags().GetString("base-url")
		if err != nil {
			log.Fatalf("get flag 'base-url': %v", err)
		}

		apiKey, err := cmd.Flags().GetString("api-key")
		if err != nil {
			log.Fatalf("get flag 'api-key': %v", err)
		}

		rawType, err := cmd.Flags().GetString("indexer-type")
		if err != nil {
			log.Fatalf("get flag 'indexer-type': %v", err)
		}

		// normalize + map string → enum
		rawType = strings.ToUpper(strings.TrimSpace(rawType))

		var idxType entity.IndexerType
		switch rawType {
		case string(entity.IndexerTypeTorznab):
			idxType = entity.IndexerTypeTorznab
		default:
			log.Fatalf("unsupported indexer type %q; valid types: %s",
				rawType, string(entity.IndexerTypeTorznab))
		}
		// now open db if there was no error
		ctx := context.Background()

		// disable this toggle later...
		db, err := utils.Open("lamplight-cli", true)
		if err != nil {
			log.Fatalf("open db: %v", err)
		}

		repo := repository.NewIndexerRepository(db)

		// construct the thing to save ..
		idx := entity.Indexer{
			Name:        name,
			BaseURL:     baseUrl,
			APIKey:      apiKey,
			IndexerType: idxType,
			Enabled:     true,
		}

		if err := repo.SaveIndexer(ctx, &idx); err != nil {
			log.Fatalf("Error: Indexer was not saved %v", err)
		}

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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addIndexerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addIndexerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
