/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// indexerCmd represents the indexer command
var indexerCmd = &cobra.Command{
	Use:   "indexer",
	Short: "Root command for controlling operations involving indexers",
	Long: `This is a root command for controlling operations involving indexers.
	This includes the following commands:
		add
		remove
		list
	`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Error: command requires a resource such as add, remove, or list.")
	},
}

func init() {
	rootCmd.AddCommand(indexerCmd)
}
