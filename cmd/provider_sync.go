package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
)

var providerSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync indexers from configured providers",
	Long: `Fetch configured indexers from enabled providers (Jackett / Prowlarr)
and add them to the Lamplight indexer table.

By default only indexers that report book category support (7000 range)
are synced. Use --all to sync every indexer regardless of categories.
If an indexer does not report any capabilities it is included by default.

This command is idempotent:
- Existing indexers are skipped
- Disabled providers are ignored
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		providerRepo := repository.NewProviderRepository(db)
		indexerRepo := repository.NewIndexerRepository(db)

		providers, err := providerRepo.FindAllProviders(ctx)
		if err != nil {
			return fmt.Errorf("load providers: %w", err)
		}
		if len(providers) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No providers configured.")
			return nil
		}

		syncAll, _ := cmd.Flags().GetBool("all")

		syncSvc := service.NewProviderSyncService(providerRepo, indexerRepo)
		report, err := syncSvc.Sync(ctx, syncAll)
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Sync complete: %d added, %d skipped\n", report.Added, report.Skipped)
		return nil
	},
}

func init() {
	providerCmd.AddCommand(providerSyncCmd)
	providerSyncCmd.Flags().BoolP("all", "a", false, "Sync all indexers regardless of category support")
}
