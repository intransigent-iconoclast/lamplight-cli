/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// listIndexersCmd represents the listIndexers command
var listIndexersCmd = &cobra.Command{
	Use:   "list",
	Short: "A brief description of your command",
	Long: `List all configured indexers in the Lamplight database.

This command prints a table of indexers with the following columns:

  - INDEX     Zero-based index for convenience when updating/deleting.
  - NAME      Logical name for the indexer (e.g. "thepiratebay").
  - PRIORITY  Integer priority used to decide which indexers to query first.
  - TYPE      Indexer type (e.g. TORZNAB).
  - ENABLED   Whether the indexer is currently enabled (yes/no).
  - BASE_URL  The base URL used to query this indexer.
  - API_KEY   (Optional) The configured API key, if --safe is provided.

By default, API keys are NOT shown. To include them in the output, use:

  lamplight indexer list --safe

Examples:

  # List all indexers (without API keys)
  lamplight indexer list

  # List all indexers and include their API keys
  lamplight indexer list --safe

You can use the INDEX column when building future commands (e.g. update or
delete) to refer to a specific indexer without having to retype its name.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewIndexerRepository(db)

		indexers, err := repo.FindAllIndexers(ctx)
		if err != nil {
			return fmt.Errorf("list indexers: %w", err)
		}

		out := cmd.OutOrStdout()

		unsafe, err := cmd.Flags().GetBool("unsafe")
		if err != nil {
			return fmt.Errorf("get flag: 'unsafe': %w", err)
		}

		if len(indexers) == 0 {
			fmt.Fprintln(out, "No indexers found. Use 'lamplight indexer add' to add one.")
			return nil
		}

		switch unsafe {
		case false:
			utils.PrintOutput(
				out,
				string(utils.INDEXER_SAFE),
				indexers,
				func(i entity.Indexer) []string {
					return []string{
						i.Name,
						i.BaseURL,
						string(i.IndexerType),
						strconv.FormatBool(i.Enabled),
						strconv.Itoa(i.Priority),
					}
				},
			)
		default:
			utils.PrintOutput(
				out,
				string(utils.INDEXER_UNSAFE),
				indexers,
				func(i entity.Indexer) []string {
					return []string{
						i.Name,
						i.BaseURL,
						string(i.IndexerType),
						strconv.FormatBool(i.Enabled),
						i.APIKey,
						strconv.Itoa(i.Priority),
					}
				},
			)
		}

		return nil
	},
}

func init() {
	indexerCmd.AddCommand(listIndexersCmd)

	listIndexersCmd.Flags().BoolP("unsafe", "u", false, "Inclusion of this flag prints passwords.")
}
