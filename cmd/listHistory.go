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
	Short: "List download history.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewHistoryRepository(db)

		entries, err := repo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}

		out := cmd.OutOrStdout()

		if len(entries) == 0 {
			fmt.Fprintln(out, "No download history yet.")
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
}
