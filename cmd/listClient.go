/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// listClientCmd represents the listClient command
var listClientCmd = &cobra.Command{
	Use:   "list",
	Short: "list all configured downloader clients.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewDownloaderRepository(db)

		// search for the downloader clients ... write a get all in the repository
		clients, err := repo.FindAllDownloaders(ctx)
		if err != nil {
			return fmt.Errorf("list clients: %w", err)
		}

		if len(clients) < 1 {
			fmt.Fprintln(cmd.OutOrStdout(), "No clients configured yet. Use 'lamplight client add' to add one.")
			return nil
		}

		// this is to get an io.Writer object (and yes i will use the word object!) since tabwriter requires one.
		out := cmd.OutOrStdout()

		unsafe, err := cmd.Flags().GetBool("unsafe")
		if err != nil {
			return fmt.Errorf("flag unsafe: %w", err)
		}

		switch unsafe {
		case false:
			utils.PrintOutput(
				out,
				string(utils.DOWNLOADER_SAFE),
				clients,
				func(d entity.Downloader) []string {
					return []string{
						d.Name,
						string(d.ClientType),
						d.Scheme,
						d.Host,
						strconv.Itoa(d.Port),
						d.BaseURL,
						d.Username,
						d.Label,
					}
				},
			)
		case true:
			utils.PrintOutput(
				out,
				string(utils.DOWNLOADER_UNSAFE),
				clients,
				func(d entity.Downloader) []string {
					return []string{
						d.Name,
						string(d.ClientType),
						d.Scheme,
						d.Host,
						strconv.Itoa(d.Port),
						d.BaseURL,
						d.Username,
						d.Password,
						d.Label,
						strconv.Itoa(d.Priority),
					}
				},
			)
		}

		return nil
	},
}

func init() {
	clientCmd.AddCommand(listClientCmd)

	listClientCmd.Flags().BoolP("unsafe", "u", false, "When flag is specified, passwords will be printed.")
}
