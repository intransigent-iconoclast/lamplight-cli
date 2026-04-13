package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var retryHistoryCmd = &cobra.Command{
	Use:   "retry <index|title>",
	Short: "re-send a stuck or failed download to Deluge.",
	Long: `re-sends the torrent to Deluge and saves the new hash so 'history sync' can track it.

select by index or title fragment:

  lamplight history retry 3
  lamplight history retry "memory of blood"

pair with --filter to narrow the list first:

  lamplight history list --filter failed
  lamplight history retry 1 --filter failed`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		filterStatus, _ := cmd.Flags().GetString("filter")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		histRepo := repository.NewHistoryRepository(db)

		var entries []entity.DownloadHistory
		if filterStatus != "" {
			entries, err = histRepo.FindByStatus(ctx, entity.DownloadStatus(filterStatus))
		} else {
			entries, err = histRepo.FindAll(ctx)
		}
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}

		target, err := resolveHistoryEntry(args[0], histRepo, entries)
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "retrying: %s\n", target.Title)

		httpClient := &http.Client{Timeout: 20 * time.Second}
		resolved, err := client.Resolve(ctx, httpClient, target.Link)
		if err != nil {
			return fmt.Errorf("resolve torrent: %w", err)
		}

		downloaderClient, _, err := createClient(ctx, db, nil)
		if err != nil {
			return fmt.Errorf("create downloader client: %w", err)
		}

		hash, err := downloaderClient.Add(ctx, resolved)
		if err != nil {
			_ = histRepo.UpdateStatus(ctx, target.ID, entity.StatusFailed)
			return fmt.Errorf("add torrent: %w", err)
		}

		if err := histRepo.UpdateStatusAndHash(ctx, target.ID, entity.StatusSnatched, hash); err != nil {
			fmt.Fprintf(out, "warning: couldn't update status: %v\n", err)
		}

		fmt.Fprintf(out, "re-sent to Deluge. status reset to snatched.\n")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(retryHistoryCmd)
	retryHistoryCmd.Flags().StringP("filter", "f", "", "filter list by status before indexing (snatched, downloading, completed, failed)")
}
