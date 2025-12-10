/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

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

		if len(indexers) == 0 {
			fmt.Fprintln(out, "No indexers found. Use 'lamplight indexer add' to add one.")
			return nil
		}

		showAPIKey, err := cmd.Flags().GetBool("safe")
		if err != nil {
			return fmt.Errorf("get flag 'safe': %w", err)
		}

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)

		// Header row
		if showAPIKey {
			fmt.Fprintln(w, "INDEX\tNAME\tPRIORITY\tTYPE\tENABLED\tBASE_URL\tAPI_KEY")
		} else {
			fmt.Fprintln(w, "INDEX\tNAME\tPRIORITY\tTYPE\tENABLED\tBASE_URL")
		}

		// Rows (INDEX is 0-based so you can map directly to slice index if you want)
		for i, idx := range indexers {
			enabled := "no"
			if idx.Enabled {
				enabled = "yes"
			}
			baseURL := strings.TrimSpace(idx.BaseURL)

			if showAPIKey {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\t%s\n",
					i,
					idx.Name,
					idx.Priority,
					string(idx.IndexerType),
					enabled,
					baseURL,
					strings.TrimSpace(idx.APIKey),
				)
			} else {
				fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
					i,
					idx.Name,
					idx.Priority,
					string(idx.IndexerType),
					enabled,
					baseURL,
				)
			}
		}

		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush writer: %w", err)
		}

		return nil
	},
}

func init() {
	indexerCmd.AddCommand(listIndexersCmd)

	listIndexersCmd.Flags().BoolP("safe", "s", false, "Will include the api key in output.")
}
