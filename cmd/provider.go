package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "manage your Prowlarr or Jackett providers.",
	Long: `configure the indexer managers lamplight pulls from.

add a provider, sync its indexers, and lamplight will automatically
know what to search. supports both Prowlarr and Jackett.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a subcommand: add, list, sync, update, delete")
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)
}
