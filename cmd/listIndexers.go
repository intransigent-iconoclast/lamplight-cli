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
	Short: "list all configured indexers.",
	Long: `show all indexers lamplight knows about. api keys are hidden by default.

  lamplight indexer list
  lamplight indexer list --unsafe   # shows api keys too`,
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
			_ = utils.PrintOutput(
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
			_ = utils.PrintOutput(
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
