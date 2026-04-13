package cmd

import (
	"fmt"
	"sort"
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

		// Filter by format/type if requested ("all" anywhere in the list skips filtering)
		if formatFilter != "" {
			types := strings.Split(strings.ToLower(strings.TrimSpace(formatFilter)), ",")
			hasAll := false
			for _, t := range types {
				if strings.TrimSpace(t) == "all" {
					hasAll = true
					break
				}
			}
			if !hasAll {
				var filtered []dao.SearchResult
				for _, r := range res {
					for _, t := range types {
						if matchesTypeFilter(r.Format, strings.TrimSpace(t)) {
							filtered = append(filtered, r)
							break
						}
					}
				}
				res = filtered
			}
		}

		// Sort results
		switch strings.ToLower(sortBy) {
		case "seeders":
			sort.SliceStable(res, func(i, j int) bool {
				si, sj := 0, 0
				if res[i].Seeders != nil {
					si = *res[i].Seeders
				}
				if res[j].Seeders != nil {
					sj = *res[j].Seeders
				}
				return si > sj
			})
		case "leechers":
			sort.SliceStable(res, func(i, j int) bool {
				li, lj := 0, 0
				if res[i].Leechers != nil {
					li = *res[i].Leechers
				}
				if res[j].Leechers != nil {
					lj = *res[j].Leechers
				}
				return li > lj
			})
		case "size":
			sort.SliceStable(res, func(i, j int) bool {
				si, sj := int64(0), int64(0)
				if res[i].SizeBytes != nil {
					si = *res[i].SizeBytes
				}
				if res[j].SizeBytes != nil {
					sj = *res[j].SizeBytes
				}
				return si > sj
			})
		case "title":
			sort.SliceStable(res, func(i, j int) bool {
				return strings.ToLower(res[i].Title) < strings.ToLower(res[j].Title)
			})
		}

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

		utils.PrintOutput(out, string(utils.SEARCH_RESULTS), res,
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

// matchesTypeFilter maps content-type aliases to the detected format values.
// "book" and "ebook" catch any prose format (epub/pdf/mobi).
// Specific formats (epub, pdf, mobi, audiobook, comic) still work as before.
func matchesTypeFilter(format, filter string) bool {
	switch filter {
	case "book", "ebook":
		return format == "epub" || format == "pdf" || format == "mobi"
	default:
		return format == filter
	}
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().IntP("indexer", "i", -1, "Search using a specific indexer.")
	searchCmd.Flags().IntP("limit", "l", 15, "Maximum number of items to return.")
	searchCmd.Flags().BoolP("books", "b", true, "Filter by book categories (default true).")
	searchCmd.Flags().StringP("sort", "s", "seeders", "Sort results by: seeders, leechers, size, title.")
	searchCmd.Flags().StringP("type", "t", "", "Filter by type: all, book (epub/pdf/mobi), audiobook, comic, epub, pdf, mobi, unknown.")
}
