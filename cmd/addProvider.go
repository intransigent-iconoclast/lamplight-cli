/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var addProviderCmd = &cobra.Command{
	Use:   "add",
	Short: "add a Prowlarr or Jackett provider.",
	Long: `point lamplight at your Prowlarr or Jackett instance.

once added, run 'lamplight provider sync' to pull all its indexers in.

  lamplight provider add --name prowlarr --type prowlarr --host localhost --port 9696 --api-key xxx
  lamplight provider add --name jackett  --type jackett  --host localhost --port 9117 --api-key xxx`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		name, _ := cmd.Flags().GetString("name")
		rawType, _ := cmd.Flags().GetString("type")
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		scheme, _ := cmd.Flags().GetString("scheme")
		apiKey, _ := cmd.Flags().GetString("api-key")

		name = strings.TrimSpace(name)
		host = strings.TrimSpace(host)
		scheme = strings.TrimSpace(scheme)
		rawType = strings.ToUpper(strings.TrimSpace(rawType))

		if name == "" {
			return fmt.Errorf("provider name cannot be empty")
		}
		if host == "" {
			return fmt.Errorf("host cannot be empty")
		}
		if port <= 0 {
			return fmt.Errorf("port must be greater than 0")
		}
		if scheme == "" {
			scheme = "http"
		}

		var providerType entity.ProviderType
		switch rawType {
		case string(entity.ProviderTypeProwlarr):
			providerType = entity.ProviderTypeProwlarr
		case string(entity.ProviderTypeJackett):
			providerType = entity.ProviderTypeJackett
		default:
			return fmt.Errorf("unsupported provider type %q (valid: PROWLARR, JACKETT)", rawType)
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewProviderRepository(db)

		provider := entity.Provider{
			Name:    name,
			Type:    providerType,
			Host:    host,
			Port:    port,
			Scheme:  scheme,
			APIKey:  apiKey,
			Enabled: true, // always enabled on add
		}

		if err := repo.SaveProvider(ctx, &provider); err != nil {
			return fmt.Errorf("save provider: %w", err)
		}

		fmt.Fprintf(
			cmd.OutOrStdout(),
			"Added provider %q (type=%s, id=%d)\n",
			provider.Name,
			provider.Type,
			provider.ID,
		)

		return nil
	},
}

func init() {
	providerCmd.AddCommand(addProviderCmd)

	addProviderCmd.Flags().StringP("name", "n", "", "logical name for the provider")
	addProviderCmd.Flags().StringP("type", "t", "", "provider type: PROWLARR or JACKETT")
	addProviderCmd.Flags().StringP("host", "H", "", "hostname or IP of the provider")
	addProviderCmd.Flags().IntP("port", "p", 0, "port the provider is listening on")
	addProviderCmd.Flags().StringP("scheme", "s", "http", "connection scheme (http or https)")
	addProviderCmd.Flags().StringP("api-key", "k", "", "API key for the provider")

	addProviderCmd.MarkFlagRequired("name")
	addProviderCmd.MarkFlagRequired("type")
	addProviderCmd.MarkFlagRequired("host")
	addProviderCmd.MarkFlagRequired("port")
}
