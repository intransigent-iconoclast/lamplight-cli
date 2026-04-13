package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var addClientCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "add a Deluge client.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		rawName := args[0]
		name := normalizeName(rawName)
		if name == "" {
			return fmt.Errorf("invalid client name")
		}

		rawClientType, err := cmd.Flags().GetString("client-type")
		if err != nil || strings.TrimSpace(rawClientType) == "" {
			return fmt.Errorf("client-type must be provided (e.g. DELUGE)")
		}

		var clientType entity.DownloaderType
		switch rawClientType {
		case string(entity.Deluge):
			clientType = entity.Deluge
		default:
			return fmt.Errorf("unsupported client type %q", rawClientType)
		}

		scheme, err := cmd.Flags().GetString("scheme")
		if err != nil || strings.TrimSpace(scheme) == "" {
			return fmt.Errorf("scheme must be provided (http or https)")
		}
		scheme = strings.ToLower(strings.TrimSpace(scheme))
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("scheme must be either http or https")
		}

		host, err := cmd.Flags().GetString("host")
		if err != nil || strings.TrimSpace(host) == "" {
			return fmt.Errorf("host is required")
		}

		port, err := cmd.Flags().GetInt("port")
		if err != nil || port <= 0 {
			return fmt.Errorf("port must be a valid number")
		}

		priority, _ := cmd.Flags().GetInt("priority")
		baseUrl, _ := cmd.Flags().GetString("base-url")
		username, _ := cmd.Flags().GetString("username")
		password, _ := cmd.Flags().GetString("password")
		label, _ := cmd.Flags().GetString("label")

		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewDownloaderRepository(db)

		client := entity.Downloader{
			Name:       name,
			ClientType: clientType,
			Host:       host,
			Scheme:     scheme,
			Port:       port,
			BaseURL:    baseUrl,
			Username:   username,
			Password:   password,
			Label:      label,
			Priority:   priority,
		}

		if err := repo.SaveClient(ctx, &client); err != nil {
			return fmt.Errorf("save client: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Added downloader client %q (id=%d)\n", name, client.ID)
		return nil
	},
}

// this function cleans user input for names
func normalizeName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') ||
			(r >= '0' && r <= '9') ||
			r == '-' {
			b.WriteRune(r)
		}
	}

	clean := b.String()
	for strings.Contains(clean, "--") {
		clean = strings.ReplaceAll(clean, "--", "-")
	}

	return strings.Trim(clean, "-")
}

func init() {
	clientCmd.AddCommand(addClientCmd)

	addClientCmd.Flags().StringP("client-type", "c", string(entity.Deluge), "Downloader type (e.g. DELUGE)")
	addClientCmd.Flags().StringP("scheme", "s", "http", "Scheme (http or https)")
	addClientCmd.Flags().StringP("host", "o", "localhost", "Downloader host")
	addClientCmd.Flags().IntP("port", "p", -1, "Downloader port")
	addClientCmd.Flags().StringP("base-url", "b", "", "Optional base URL")
	addClientCmd.Flags().StringP("username", "u", "", "Downloader username")
	addClientCmd.Flags().StringP("password", "w", "", "Downloader password")
	addClientCmd.Flags().StringP("label", "l", "", "Label for added torrents")
	addClientCmd.Flags().IntP("priority", "d", 42, "Downloader priority (lower = higher priority)")

	addClientCmd.MarkFlagRequired("client-type")
	addClientCmd.MarkFlagRequired("host")
	addClientCmd.MarkFlagRequired("port")
}
