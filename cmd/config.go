package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and update lamplight settings.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a subcommand: get, set")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
