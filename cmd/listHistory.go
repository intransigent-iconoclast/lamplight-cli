package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var listHistoryCmd = &cobra.Command{
	Use:   "list [search]",
	Short: "show your download history.",
	Long: `list all downloads. filter by status or search by title.

  lamplight history list
  lamplight history list --filter failed
  lamplight history list "memory of blood"
  lamplight history list "fowler" --filter completed`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		filterStatus, _ := cmd.Flags().GetString("filter")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewHistoryRepository(db)

		var entries []entity.DownloadHistory
		if filterStatus != "" {
			entries, err = repo.FindByStatus(ctx, entity.DownloadStatus(filterStatus))
		} else {
			entries, err = repo.FindAll(ctx)
		}
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}

		// fuzzy title filter
		if len(args) > 0 {
			query := strings.ToLower(args[0])
			var matched []entity.DownloadHistory
			for _, e := range entries {
				if strings.Contains(strings.ToLower(e.Title), query) {
					matched = append(matched, e)
				}
			}
			entries = matched
		}

		out := cmd.OutOrStdout()

		if len(entries) == 0 {
			fmt.Fprintln(out, "no entries found.")
			return nil
		}

		utils.PrintOutput(out, string(utils.HISTORY), entries, func(e entity.DownloadHistory) []string {
			return []string{
				utils.CleanString(e.Title),
				e.IndexerName,
				e.DownloaderName,
				utils.BytesToMb(int(e.SizeBytes)),
				string(e.Status),
				e.DownloadedAt.Format("2006-01-02 15:04"),
			}
		})

		return nil
	},
}

func init() {
	historyCmd.AddCommand(listHistoryCmd)
	listHistoryCmd.Flags().StringP("filter", "f", "", "filter by status: snatched, downloading, completed, failed")
}
