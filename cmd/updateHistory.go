package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var validStatuses = []entity.DownloadStatus{
	entity.StatusSnatched,
	entity.StatusDownloading,
	entity.StatusCompleted,
	entity.StatusFailed,
}

var updateHistoryCmd = &cobra.Command{
	Use:   "update <index|title>",
	Short: "manually fix the status of a download.",
	Long: `set the status of a download entry by index or title fragment.

  lamplight history update 3 --status failed
  lamplight history update "memory of blood" --status failed

valid statuses: snatched, downloading, completed, failed`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		newStatus, _ := cmd.Flags().GetString("status")
		if newStatus == "" {
			return fmt.Errorf("--status is required (snatched, downloading, completed, failed)")
		}

		status := entity.DownloadStatus(newStatus)
		valid := false
		for _, s := range validStatuses {
			if status == s {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid status '%s' — must be one of: snatched, downloading, completed, failed", newStatus)
		}

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

		target, err := resolveHistoryEntry(args[0], repo, entries)
		if err != nil {
			return err
		}

		if err := repo.UpdateStatus(ctx, target.ID, status); err != nil {
			return fmt.Errorf("update status: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "updated '%s' → %s\n", target.Title, status)
		return nil
	},
}

func init() {
	historyCmd.AddCommand(updateHistoryCmd)
	updateHistoryCmd.Flags().StringP("status", "s", "", "new status: snatched, downloading, completed, failed")
	updateHistoryCmd.Flags().StringP("filter", "f", "", "filter list by status before indexing (snatched, downloading, completed, failed)")
}
