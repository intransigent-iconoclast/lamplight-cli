package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "view and manage your download history.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a subcommand: list, sync, update, retry, clear")
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
