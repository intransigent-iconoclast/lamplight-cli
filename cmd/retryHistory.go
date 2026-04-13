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
	Use:   "retry <index>",
	Short: "Re-send a download to the client.",
	Long:  "Re-requests a download that got stuck or failed. Resets status to snatched.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()

		index, err := strconv.Atoi(args[0])
		if err != nil || index <= 0 {
			return fmt.Errorf("invalid index '%s': must be a positive number", args[0])
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		histRepo := repository.NewHistoryRepository(db)

		entries, err := histRepo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}

		if index > len(entries) {
			return fmt.Errorf("index %d out of range (have %d entries)", index, len(entries))
		}

		target := entries[index-1]

		fmt.Fprintf(out, "Retrying: %s\n", target.Title)

		httpClient := &http.Client{Timeout: 20 * time.Second}
		resolved, err := client.Resolve(ctx, httpClient, target.Link)
		if err != nil {
			return fmt.Errorf("resolve torrent: %w", err)
		}

		downloaderClient, _, err := createClient(ctx, db, nil)
		if err != nil {
			return fmt.Errorf("create downloader client: %w", err)
		}

		if err := downloaderClient.Add(ctx, resolved); err != nil {
			// mark as failed so the user knows it didn't go through
			_ = histRepo.UpdateStatus(ctx, target.ID, entity.StatusFailed)
			return fmt.Errorf("add torrent: %w", err)
		}

		if err := histRepo.UpdateStatus(ctx, target.ID, entity.StatusSnatched); err != nil {
			fmt.Fprintf(out, "warning: couldn't reset status: %v\n", err)
		}

		fmt.Fprintf(out, "Re-sent to client. Status reset to snatched.\n")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(retryHistoryCmd)
}
