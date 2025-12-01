/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// deleteIndexerCmd represents the deleteIndexer command
var deleteIndexerCmd = &cobra.Command{
	Use:   "deleteIndexer",
	Short: "A brief description of your command",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("deleteIndexer called")
	},
}

func init() {
	rootCmd.AddCommand(deleteIndexerCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// deleteIndexerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// deleteIndexerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
