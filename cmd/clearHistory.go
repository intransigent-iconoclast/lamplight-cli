package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var clearHistoryCmd = &cobra.Command{
	Use:   "clear",
	Short: "wipe all download history.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewHistoryRepository(db)

		if err := repo.DeleteAll(ctx); err != nil {
			return fmt.Errorf("clear history: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Download history cleared.")
		return nil
	},
}

func init() {
	historyCmd.AddCommand(clearHistoryCmd)
}
