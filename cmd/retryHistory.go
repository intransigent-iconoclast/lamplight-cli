package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
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

		histRepo := repository.NewHistoryRepository(db)

		// --- retry all failed ---
		if allFailed {
			entries, err := histRepo.FindByStatus(ctx, entity.StatusFailed)
			if err != nil {
				return fmt.Errorf("load failed entries: %w", err)
			}
			if len(entries) == 0 {
				fmt.Fprintln(out, "no failed downloads to retry.")
				return nil
			}

			downloaderClient, _, err := createClient(ctx, db, nil)
			if err != nil {
				return fmt.Errorf("connect to downloader: %w", err)
			}

			httpClient := &http.Client{Timeout: 20 * time.Second}
			for _, entry := range entries {
				resolved, err := client.Resolve(ctx, httpClient, entry.Link)
				if err != nil {
					fmt.Fprintf(out, "  skip  %s — resolve: %v\n", utils.SmartTruncate(entry.Title, 50), err)
					continue
				}
				hash, err := downloaderClient.Add(ctx, resolved)
				if err != nil {
					fmt.Fprintf(out, "  fail  %s — %v\n", utils.SmartTruncate(entry.Title, 50), err)
					continue
				}
				if err := histRepo.UpdateStatusAndHash(ctx, entry.ID, entity.StatusSnatched, hash); err != nil {
					fmt.Fprintf(out, "  warn  %s — couldn't update status: %v\n", utils.SmartTruncate(entry.Title, 50), err)
					continue
				}
				fmt.Fprintf(out, "  ok    %s\n", utils.SmartTruncate(entry.Title, 50))
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

		entries, err := histRepo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}
		if index > len(entries) {
			return fmt.Errorf("index %d out of range (have %d entries)", index, len(entries))
		}

		target := entries[index-1]
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
	retryHistoryCmd.Flags().Bool("all-failed", false, "retry every failed download at once")
}
