package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "manage your downloader clients (Deluge).",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a subcommand: add, list, update, delete")
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)
}
