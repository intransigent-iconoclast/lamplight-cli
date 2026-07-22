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

		downloadSvc := service.NewDownloadService(
			repository.NewHistoryRepository(db),
			repository.NewDownloaderRepository(db),
			&http.Client{Timeout: 20 * time.Second},
		)

		result, err := downloadSvc.CancelByIndex(ctx, index, deleteData)
		if err != nil {
			return err
		}

		fmt.Fprintf(out, "cancelling: %s\n", result.Title)
		switch {
		case !result.HadHash:
			fmt.Fprintln(out, "  no torrent hash — skipping Deluge removal")
		case result.Warning != "":
			fmt.Fprintf(out, "  warn  %s\n", result.Warning)
		case deleteData:
			fmt.Fprintln(out, "  removed from Deluge (files deleted)")
		default:
			fmt.Fprintln(out, "  removed from Deluge (files kept)")
		}
		fmt.Fprintln(out, "  removed from history")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(cancelHistoryCmd)
	cancelHistoryCmd.Flags().Bool("delete-data", false, "also delete the downloaded files from disk")
}
