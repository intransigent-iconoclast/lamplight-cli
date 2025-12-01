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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// indexerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// indexerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
