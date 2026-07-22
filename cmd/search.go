package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "search across all your configured indexers.",
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
		sortBy, _ := cmd.Flags().GetString("sort")
		formatFilter, _ := cmd.Flags().GetString("type")

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

		res = service.FilterByFormat(res, formatFilter)
		service.SortResults(res, service.SortKey(strings.ToLower(sortBy)))

		out := cmd.OutOrStdout()

		if len(res) == 0 {
			fmt.Fprintln(out, "No results.")
			return nil
		}

		// non-title columns take ~68 chars, title gets the rest (min 20, max 80)
		termW := utils.TerminalWidth()
		titleMax := termW - 68
		if titleMax < 20 {
			titleMax = 20
		}
		if titleMax > 80 {
			titleMax = 80
		}

		_ = utils.PrintOutput(out, string(utils.SEARCH_RESULTS), res,
			func(d dao.SearchResult) []string {
				size := "?"
				if d.SizeBytes != nil {
					size = utils.BytesToMb(int(*d.SizeBytes))
				}
				seeders := "?"
				if d.Seeders != nil {
					seeders = strconv.Itoa(*d.Seeders)
				}
				leechers := "?"
				if d.Leechers != nil {
					leechers = strconv.Itoa(*d.Leechers)
				}
				return []string{
					utils.SmartTruncate(utils.CleanString(d.Title), titleMax),
					d.Format,
					utils.CleanIndexerName(d.IndexerName),
					size,
					seeders,
					leechers,
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
	searchCmd.Flags().StringP("sort", "s", "seeders", "Sort results by: seeders, leechers, size, title.")
	searchCmd.Flags().StringP("type", "t", "", "Filter by type: all, book (epub/pdf/mobi), audiobook, comic, epub, pdf, mobi, unknown.")
}
