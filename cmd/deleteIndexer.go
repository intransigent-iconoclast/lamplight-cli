/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// deleteIndexerCmd represents the deleteIndexer command
var deleteIndexerCmd = &cobra.Command{
	Use:   "delete",
	Short: "A brief description of your command",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		// input validation since we CAN'T use MarkFlagRequired (we allow multiple options conceptually)
		useIndex := cmd.Flags().Changed("index")
		useName := cmd.Flags().Changed("name")

		count := 0
		if useIndex {
			count++
		}
		if useName {
			count++
		}

		if count == 0 {
			// show help and exit with an error
			return fmt.Errorf("you must specify exactly one of --index or --name")
		}
		if count > 1 {
			return fmt.Errorf("please specify only one of --index or --name (got multiple)")
		}

		ctx := cmd.Context()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		repo := repository.NewIndexerRepository(db)

		// this is who we're gonna delete bad boyyyyy
		var target *entity.Indexer

		switch {
		case useIndex:
			idx, err := cmd.Flags().GetInt("index")
			if err != nil {
				return fmt.Errorf("get flag 'index': %w", err)
			}
			if idx < 0 {
				return fmt.Errorf("index must be >= 0 (got %d)", idx)
			}

			indexers, err := repo.FindAllIndexers(ctx)
			if err != nil {
				return fmt.Errorf("load indexers: %w", err)
			}
			if idx >= len(indexers) {
				return fmt.Errorf("index %d out of range (have %d indexers)", idx, len(indexers))
			}
			target = &indexers[idx]

		case useName:
			name, err := cmd.Flags().GetString("name")
			if err != nil {
				return fmt.Errorf("get flag 'name': %w", err)
			}
			target, err = repo.FindByName(ctx, name)
			if err != nil {
				return fmt.Errorf("find by name: %w", err)
			}
		}

		if target == nil {
			return fmt.Errorf("no indexer found matching given selector")
		}

		if err := repo.DeleteIndexerById(ctx, target.ID); err != nil {
			return fmt.Errorf("delete indexer: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted indexer %q (id=%d)\n", target.Name, target.ID)
		return nil
	},
}

func init() {
	indexerCmd.AddCommand(deleteIndexerCmd)

	// have to use -1 because IntP requires a value ...
	deleteIndexerCmd.Flags().IntP("index", "i", -1, "Allow")
	deleteIndexerCmd.Flags().StringP("name", "n", "", "Name of the indexer")
}
