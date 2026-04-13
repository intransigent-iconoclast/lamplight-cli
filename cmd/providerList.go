package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var listProvidersCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured providers.",
	Long: `List all configured indexer providers (Prowlarr or Jackett).

This command prints a table of providers with the following columns:

  - INDEX    Zero-based index for convenience.
  - NAME     Logical provider name (e.g. "prowlarr-local").
  - TYPE     Provider type (PROWLARR or JACKETT).
  - HOST     Hostname or IP.
  - PORT     Port number.
  - SCHEME   http or https.
  - ENABLED  Whether the provider is enabled.

By default, API keys are NOT shown. To include them, use:

  lamplight provider list --unsafe
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewProviderRepository(db)

		providers, err := repo.FindAllProviders(ctx)
		if err != nil {
			return fmt.Errorf("list providers: %w", err)
		}

		out := cmd.OutOrStdout()

		unsafe, err := cmd.Flags().GetBool("unsafe")
		if err != nil {
			return fmt.Errorf("get flag 'unsafe': %w", err)
		}

		if len(providers) == 0 {
			fmt.Fprintln(out, "No providers found. Use 'lamplight provider add' to add one.")
			return nil
		}

		switch unsafe {
		case false:
			utils.PrintOutput(
				out,
				string(utils.PROVIDER_SAFE),
				providers,
				func(p entity.Provider) []string {
					return []string{
						p.Name,
						string(p.Type),
						p.Host,
						strconv.Itoa(p.Port),
						p.Scheme,
						strconv.FormatBool(p.Enabled),
					}
				},
			)
		default:
			utils.PrintOutput(
				out,
				string(utils.PROVIDER_UNSAFE),
				providers,
				func(p entity.Provider) []string {
					return []string{
						p.Name,
						string(p.Type),
						p.Host,
						strconv.Itoa(p.Port),
						p.Scheme,
						p.APIKey,
						strconv.FormatBool(p.Enabled),
					}
				},
			)
		}

		return nil
	},
}

func init() {
	providerCmd.AddCommand(listProvidersCmd)
	listProvidersCmd.Flags().BoolP("unsafe", "u", false, "Include API keys in output.")
}
