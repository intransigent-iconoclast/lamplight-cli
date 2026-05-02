package cmd

import (
	"fmt"
	"strconv"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var deleteClientCmd = &cobra.Command{
	Use:   "delete <index>",
	Short: "Remove a downloader client by index.",
	Long: `Remove a downloader client. Use the index shown in 'lamplight client list'.

  lamplight client delete 1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		idx, err := strconv.Atoi(args[0])
		if err != nil || idx <= 0 {
			return fmt.Errorf("invalid index %q — use the number shown in 'lamplight client list'", args[0])
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewDownloaderRepository(db)

		clients, err := repo.FindAllDownloaders(ctx)
		if err != nil {
			return fmt.Errorf("load clients: %w", err)
		}

		if idx > len(clients) {
			return fmt.Errorf("index %d out of range (found %d clients)", idx, len(clients))
		}

		client := clients[idx-1]

		if err := repo.DeleteByName(ctx, client.Name); err != nil {
			return fmt.Errorf("delete client: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted downloader client %q\n", client.Name)
		return nil
	},
}

func init() {
	clientCmd.AddCommand(deleteClientCmd)
}
