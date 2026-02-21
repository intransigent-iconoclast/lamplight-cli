package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for content across configured indexers.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			return fmt.Errorf("please provide a search query")
		}

		indexer, _ := cmd.Flags().GetInt("indexer")
		limit, _ := cmd.Flags().GetInt("limit")
		books, _ := cmd.Flags().GetBool("books")

		var searchRequest dao.SearchRequest
		searchRequest.Query = query

		if limit > 0 {
			searchRequest.Limit = limit
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		repo := repository.NewIndexerRepository(db)

		if indexer > -1 {
			indexers, err := repo.FindAllIndexers(ctx)
			if err != nil {
				return fmt.Errorf("load indexers: %w", err)
			}
			if len(indexers) == 0 {
				return fmt.Errorf("no indexers defined; use 'lamplight indexer add' first")
			}
			if indexer < 0 || indexer >= len(indexers) {
				return fmt.Errorf("indexer index %d out of range (have %d indexers)", indexer, len(indexers))
			}

			searchRequest.IndexerName = indexers[indexer].Name
		}

		torznabClient := client.NewTorznabClient(nil)
		torznabBackend := service.NewTorznabBackend(torznabClient)

		searchService := service.NewSearchService(repo, []service.SearchBackend{torznabBackend})

		var criteria dao.FilterCriteria
		if books {
			criteria = dao.FilterCriteria{
				AllowedCategories: constants.BookCategories,
			}
		}

		res, err := searchService.Search(ctx, searchRequest, criteria)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		out := cmd.OutOrStdout()

		if len(res) == 0 {
			fmt.Fprintln(out, "No results.")
			return nil
		}

		utils.PrintOutput(out, string(utils.SEARCH_RESULTS), res,
			func(d dao.SearchResult) []string {
				return []string{
					utils.CleanString(d.Title),
					d.IndexerName,
					utils.BytesToMb(int(*d.SizeBytes)),
					strconv.Itoa(*d.Seeders),
					strconv.Itoa(*d.Seeders),
				}
			})

		// Cache results
		cache := repository.NewCacheRepository(db)
		if err := cache.AddResultsToCache(ctx, &res); err != nil {
			fmt.Fprintf(out, "warning: failed to cache results: %v\n", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntP("indexer", "i", -1, "Search using a specific indexer.")
	searchCmd.Flags().IntP("limit", "l", 15, "Maximum number of items to return.")
	searchCmd.Flags().BoolP("books", "b", true, "Filter by book categories (default true).")
}
