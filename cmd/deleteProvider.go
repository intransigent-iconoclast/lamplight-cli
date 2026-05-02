package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var deleteProviderCmd = &cobra.Command{
	Use:   "delete <index>",
	Short: "Remove a provider by index.",
	Long: `Remove a provider. Use the index shown in 'lamplight provider list'.

  lamplight provider delete 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		idx, err := strconv.Atoi(args[0])
		if err != nil || idx <= 0 {
			return fmt.Errorf("invalid index %q — use the number shown in 'lamplight provider list'", args[0])
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewProviderRepository(db)

		providers, err := repo.FindAllProviders(ctx)
		if err != nil {
			return fmt.Errorf("load providers: %w", err)
		}

		if idx > len(providers) {
			return fmt.Errorf("index %d out of range (found %d providers)", idx, len(providers))
		}

		provider := providers[idx-1]

		if err := repo.DeleteByName(ctx, provider.Name); err != nil {
			return fmt.Errorf("delete provider: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted provider %q\n", provider.Name)
		return nil
	},
}

func init() {
	providerCmd.AddCommand(deleteProviderCmd)
}
