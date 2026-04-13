/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var updateProviderCmd = &cobra.Command{
	Use:   "update <index>",
	Short: "Update an existing provider.",
	Long: `Update fields on an existing provider.

The provider is selected using the INDEX shown in:

  lamplight provider list

Only fields explicitly provided will be updated.

Enable or disable the provider using flags:

  --enable
  --disable
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		idx, err := strconv.Atoi(args[0])
		if err != nil || idx <= 0 {
			return fmt.Errorf("invalid index %q", args[0])
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

		name, _ := cmd.Flags().GetString("name")
		rawType, _ := cmd.Flags().GetString("type")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		scheme, _ := cmd.Flags().GetString("scheme")
		apiKey, _ := cmd.Flags().GetString("api-key")

		enable, _ := cmd.Flags().GetBool("enable")
		disable, _ := cmd.Flags().GetBool("disable")

		if enable && disable {
			return fmt.Errorf("cannot use --enable and --disable together")
		}

		if strings.TrimSpace(name) != "" {
			provider.Name = strings.TrimSpace(name)
		}

		if strings.TrimSpace(rawType) != "" {
			rawType = strings.ToUpper(strings.TrimSpace(rawType))
			switch rawType {
			case string(entity.ProviderTypeProwlarr):
				provider.Type = entity.ProviderTypeProwlarr
			case string(entity.ProviderTypeJackett):
				provider.Type = entity.ProviderTypeJackett
			default:
				return fmt.Errorf("unsupported provider type %q", rawType)
			}
		}

		if strings.TrimSpace(host) != "" {
			provider.Host = strings.TrimSpace(host)
		}

		if port > 0 {
			provider.Port = port
		}

		if strings.TrimSpace(scheme) != "" {
			provider.Scheme = strings.TrimSpace(scheme)
		}

		if apiKey != "" {
			provider.APIKey = apiKey
		}

		if enable {
			provider.Enabled = true
		}

		if disable {
			provider.Enabled = false
		}

		if err := repo.UpdateProvider(ctx, &provider); err != nil {
			return fmt.Errorf("update provider: %w", err)
		}

		fmt.Fprintf(
			cmd.OutOrStdout(),
			"Updated provider %q (id=%d)\n",
			provider.Name,
			provider.ID,
		)

		return nil
	},
}

func init() {
	providerCmd.AddCommand(updateProviderCmd)

	updateProviderCmd.Flags().StringP("name", "n", "", "update provider name")
	updateProviderCmd.Flags().StringP("type", "t", "", "update provider type (PROWLARR or JACKETT)")
	updateProviderCmd.Flags().StringP("host", "H", "", "update host")
	updateProviderCmd.Flags().IntP("port", "p", 0, "update port")
	updateProviderCmd.Flags().StringP("scheme", "s", "", "update scheme (http or https)")
	updateProviderCmd.Flags().StringP("api-key", "k", "", "update API key")
	updateProviderCmd.Flags().BoolP("enable", "e", false, "enable provider")
	updateProviderCmd.Flags().BoolP("disable", "d", false, "disable provider")
}
