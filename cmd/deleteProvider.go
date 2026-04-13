package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var deleteProviderCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "remove a provider by name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		name := args[0]

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewProviderRepository(db)

		if err := repo.DeleteByName(ctx, name); err != nil {
			return fmt.Errorf("delete provider: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted provider %q\n", name)
		return nil
	},
}

func init() {
	providerCmd.AddCommand(deleteProviderCmd)
}
