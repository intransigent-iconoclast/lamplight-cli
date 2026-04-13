package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var listHistoryCmd = &cobra.Command{
	Use:   "list",
	Short: "show your download history.",
	Long: `list all downloads. filter by status to find stuck or failed ones.

  lamplight history list
  lamplight history list --filter failed
  lamplight history list --filter snatched`,
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

		out := cmd.OutOrStdout()

		if len(entries) == 0 {
			if filterStatus != "" {
				fmt.Fprintf(out, "no entries with status '%s'\n", filterStatus)
			} else {
				fmt.Fprintln(out, "no download history yet.")
			}
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
