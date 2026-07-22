package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
)

var retryHistoryCmd = &cobra.Command{
	Use:   "retry [index]",
	Short: "re-send a stuck or failed download to Deluge.",
	Long: `re-sends a torrent to Deluge and saves the new hash so 'history sync' can track it.

retry a single entry by index:

  lamplight history list --filter failed
  lamplight history retry 3

or retry everything that's failed in one shot:

  lamplight history retry --all-failed`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		allFailed, _ := cmd.Flags().GetBool("all-failed")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		downloadSvc := service.NewDownloadService(
			repository.NewHistoryRepository(db),
			repository.NewDownloaderRepository(db),
			&http.Client{Timeout: 20 * time.Second},
		)

		// --- retry all failed ---
		if allFailed {
			results, err := downloadSvc.RetryAllFailed(ctx)
			if err != nil {
				return err
			}
			if len(results) == 0 {
				fmt.Fprintln(out, "no failed downloads to retry.")
				return nil
			}
			for _, r := range results {
				if r.Warning != "" {
					fmt.Fprintf(out, "  warn  %s — %s\n", utils.SmartTruncate(r.Title, 50), r.Warning)
				} else {
					fmt.Fprintf(out, "  ok    %s\n", utils.SmartTruncate(r.Title, 50))
				}
			}
			return nil
		}

		// --- retry single by index ---
		if len(args) == 0 {
			return fmt.Errorf("provide an index or use --all-failed")
		}

		index, err := strconv.Atoi(args[0])
		if err != nil || index <= 0 {
			return fmt.Errorf("invalid index '%s': must be a positive number", args[0])
		}

		result, err := downloadSvc.RetryByIndex(ctx, index)
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "retrying: %s\n", result.Title)
		if result.Warning != "" {
			fmt.Fprintf(out, "warning: %s\n", result.Warning)
		}
		fmt.Fprintf(out, "re-sent to Deluge. status reset to snatched.\n")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(retryHistoryCmd)
	retryHistoryCmd.Flags().Bool("all-failed", false, "retry every failed download at once")
}
