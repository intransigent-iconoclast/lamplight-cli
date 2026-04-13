package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var updateIndexerCmd = &cobra.Command{
	Use:   "update <index>",
	Short: "Update an existing indexer.",
	Long: `Update fields on an existing indexer.

The indexer is selected using the INDEX shown in:

  lamplight indexer list

Only flags explicitly provided will be updated.

Enable or disable the indexer with:

  --enable
  --disable
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		idx, err := strconv.Atoi(args[0])
		if err != nil || idx <= 0 {
			return fmt.Errorf("invalid index %q: must be a positive number", args[0])
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewIndexerRepository(db)

		indexers, err := repo.FindAllIndexers(ctx)
		if err != nil {
			return fmt.Errorf("load indexers: %w", err)
		}

		if idx > len(indexers) {
			return fmt.Errorf("index %d out of range (found %d indexers)", idx, len(indexers))
		}

		indexer := indexers[idx-1]

		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")
		if enable && disable {
			return fmt.Errorf("cannot use --enable and --disable together")
		}

		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("name cannot be empty")
			}
			indexer.Name = strings.TrimSpace(name)
		}

		if cmd.Flags().Changed("base-url") {
			baseURL, _ := cmd.Flags().GetString("base-url")
			if strings.TrimSpace(baseURL) == "" {
				return fmt.Errorf("base-url cannot be empty")
			}
			indexer.BaseURL = strings.TrimSpace(baseURL)
		}

		if cmd.Flags().Changed("api-key") {
			apiKey, _ := cmd.Flags().GetString("api-key")
			indexer.APIKey = apiKey
		}

		if cmd.Flags().Changed("priority") {
			priority, _ := cmd.Flags().GetInt("priority")
			indexer.Priority = priority
		}

		if enable {
			indexer.Enabled = true
		}
		if disable {
			indexer.Enabled = false
		}

		if err := repo.UpdateIndexer(ctx, &indexer); err != nil {
			return fmt.Errorf("update indexer: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated indexer %q (id=%d)\n", indexer.Name, indexer.ID)
		return nil
	},
}

func init() {
	indexerCmd.AddCommand(updateIndexerCmd)

	updateIndexerCmd.Flags().StringP("name", "n", "", "update indexer name")
	updateIndexerCmd.Flags().StringP("base-url", "u", "", "update base URL")
	updateIndexerCmd.Flags().StringP("api-key", "k", "", "update API key")
	updateIndexerCmd.Flags().IntP("priority", "r", 0, "update priority (lower = higher priority)")
	updateIndexerCmd.Flags().BoolP("enable", "e", false, "enable this indexer")
	updateIndexerCmd.Flags().BoolP("disable", "d", false, "disable this indexer")
}
