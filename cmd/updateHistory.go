package cmd

import (
	"fmt"
	"strconv"

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
	Use:   "update <index>",
	Short: "manually fix the status of a download.",
	Long: `set the status on a download by its index number.

use --filter to narrow the list first so the numbers stay small:

  lamplight history list --filter failed
  lamplight history update 1 --filter failed --status snatched

valid statuses: snatched, downloading, completed, failed`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		index, err := strconv.Atoi(args[0])
		if err != nil || index <= 0 {
			return fmt.Errorf("invalid index '%s': must be a positive number", args[0])
		}

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

		if index > len(entries) {
			return fmt.Errorf("index %d out of range (showing %d entries)", index, len(entries))
		}

		target := entries[index-1]

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
