package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var updateClientCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "update an existing downloader client.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		rawName := args[0]
		name := normalizeName(rawName)
		if name == "" {
			return fmt.Errorf("invalid client name")
		}

		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewDownloaderRepository(db)

		client, err := repo.FindByName(ctx, name)
		if err != nil {
			return fmt.Errorf("find client: %w", err)
		}
		if client == nil {
			return fmt.Errorf("client %q not found", name)
		}

		if cmd.Flags().Changed("scheme") {
			scheme, _ := cmd.Flags().GetString("scheme")
			scheme = strings.ToLower(strings.TrimSpace(scheme))
			if scheme != "http" && scheme != "https" {
				return fmt.Errorf("scheme must be either http or https")
			}
			client.Scheme = scheme
		}

		if cmd.Flags().Changed("host") {
			host, _ := cmd.Flags().GetString("host")
			if strings.TrimSpace(host) == "" {
				return fmt.Errorf("host cannot be empty")
			}
			client.Host = host
		}

		if cmd.Flags().Changed("port") {
			port, _ := cmd.Flags().GetInt("port")
			if port <= 0 {
				return fmt.Errorf("port must be a valid number")
			}
			client.Port = port
		}

		if cmd.Flags().Changed("username") {
			username, _ := cmd.Flags().GetString("username")
			client.Username = username
		}

		if cmd.Flags().Changed("password") {
			password, _ := cmd.Flags().GetString("password")
			client.Password = password
		}

		if cmd.Flags().Changed("label") {
			label, _ := cmd.Flags().GetString("label")
			client.Label = label
		}

		if cmd.Flags().Changed("priority") {
			priority, _ := cmd.Flags().GetInt("priority")
			client.Priority = priority
		}

		if err := repo.Update(ctx, client); err != nil {
			return fmt.Errorf("update client: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updated downloader client %q\n", name)
		return nil
	},
}

func init() {
	clientCmd.AddCommand(updateClientCmd)

	updateClientCmd.Flags().StringP("scheme", "s", "", "Scheme (http or https)")
	updateClientCmd.Flags().StringP("host", "o", "", "Downloader host")
	updateClientCmd.Flags().IntP("port", "p", 0, "Downloader port")
	updateClientCmd.Flags().StringP("base-url", "b", "", "Optional base URL")
	updateClientCmd.Flags().StringP("username", "u", "", "Downloader username")
	updateClientCmd.Flags().StringP("password", "w", "", "Downloader password")
	updateClientCmd.Flags().StringP("label", "l", "", "Label for added torrents")
	updateClientCmd.Flags().IntP("priority", "d", 0, "Downloader priority")
}
