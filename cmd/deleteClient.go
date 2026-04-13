package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var deleteClientCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "remove a downloader client by name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		name := args[0]

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewDownloaderRepository(db)

		if err := repo.DeleteByName(ctx, name); err != nil {
			return fmt.Errorf("delete client: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted downloader client %q\n", name)
		return nil
	},
}

func init() {
	clientCmd.AddCommand(deleteClientCmd)
}
