package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var cancelHistoryCmd = &cobra.Command{
	Use:   "cancel <index>",
	Short: "cancel a download and remove it from Deluge.",
	Long: `removes a torrent from Deluge and deletes it from your history.

  lamplight history list
  lamplight history cancel 3

by default the downloaded files are left on disk. use --delete-data to remove them too:

  lamplight history cancel 3 --delete-data`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		deleteData, _ := cmd.Flags().GetBool("delete-data")

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
		fmt.Fprintf(out, "cancelling: %s\n", target.Title)

		// if we have a hash, tell Deluge to remove it
		if target.TorrentHash != "" {
			downloaderClient, _, err := createClient(ctx, db, nil)
			if err != nil {
				return fmt.Errorf("connect to deluge: %w", err)
			}

			if err := downloaderClient.Remove(ctx, target.TorrentHash, deleteData); err != nil {
				// don't bail — if Deluge already finished or the torrent is gone, still clean up history
				fmt.Fprintf(out, "  warn  couldn't remove from Deluge: %v\n", err)
			} else {
				if deleteData {
					fmt.Fprintln(out, "  removed from Deluge (files deleted)")
				} else {
					fmt.Fprintln(out, "  removed from Deluge (files kept)")
				}
			}
		} else {
			fmt.Fprintln(out, "  no torrent hash — skipping Deluge removal")
		}

		if err := histRepo.Delete(ctx, target.ID); err != nil {
			return fmt.Errorf("remove from history: %w", err)
		}

		fmt.Fprintln(out, "  removed from history")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(cancelHistoryCmd)
	cancelHistoryCmd.Flags().Bool("delete-data", false, "also delete the downloaded files from disk")
}
