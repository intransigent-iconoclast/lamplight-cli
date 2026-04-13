package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var configGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Show current lamplight config.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewLibraryConfigRepository(db)
		cfg, err := repo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "library-path  %s\n", cfg.LibraryPath)
		fmt.Fprintf(out, "template      %s\n", cfg.Template)

		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd)
}
