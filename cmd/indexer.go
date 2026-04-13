package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// indexerCmd represents the indexer command
var indexerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "manage your configured indexers.",
	Long: `add, remove, update, and list the torznab indexers lamplight queries.

most of the time you won't touch this directly — 'provider sync' handles it.
but if you need to manually add or tweak an indexer, this is the place.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a subcommand: add, list, update, delete")
	},
}

func init() {
	rootCmd.AddCommand(indexerCmd)
}
