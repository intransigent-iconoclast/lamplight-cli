package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var listHistoryCmd = &cobra.Command{
	Use:   "list [search]",
	Short: "show your download history.",
	Long: `list all downloads. filter by status or search by title.

the index shown is always the global index — use it directly with retry or update.

  lamplight history list
  lamplight history list --filter failed
  lamplight history list "memory of blood"
  lamplight history list "fowler" --filter completed`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		filterStatus, _ := cmd.Flags().GetString("filter")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewHistoryRepository(db)

		// always load full list so indices stay accurate for retry/update
		all, err := repo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}

		type indexed struct {
			globalIdx int
			entry     entity.DownloadHistory
		}

		var rows []indexed
		for i, e := range all {
			if filterStatus != "" && string(e.Status) != filterStatus {
				continue
			}
			if len(args) > 0 && !strings.Contains(strings.ToLower(e.Title), strings.ToLower(args[0])) {
				continue
			}
			rows = append(rows, indexed{i + 1, e})
		}

		out := cmd.OutOrStdout()

		if len(rows) == 0 {
			fmt.Fprintln(out, "no entries found.")
			return nil
		}

		entries := make([]entity.DownloadHistory, len(rows))
		indices := make([]int, len(rows))
		for i, r := range rows {
			entries[i] = r.entry
			indices[i] = r.globalIdx
		}

		_ = utils.PrintOutputWithIndices(out, string(utils.HISTORY), entries, indices, func(e entity.DownloadHistory) []string {
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
