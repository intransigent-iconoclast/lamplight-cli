package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/render"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup <author>",
	Short: "look up an author's bibliography and see what you're missing.",
	Long: `look up an author and see their books grouped by series, with a check
next to the ones you already have.

  lamplight lookup "becky chambers"
  lamplight lookup "becky chambers" --plain     # no color/borders (good for pipes)
  lamplight lookup "becky chambers" --json      # machine-readable
  lamplight lookup --get 3                       # search & download book #3 from the last lookup

ownership is matched against your download history for now — connect a library
server (komga/kavita/calibre) later for a complete picture.`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()

		get, _ := cmd.Flags().GetInt("get")
		plain, _ := cmd.Flags().GetBool("plain")
		asJSON, _ := cmd.Flags().GetBool("json")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		if get > 0 {
			return lookupGet(ctx, db, out, get)
		}

		query := strings.TrimSpace(strings.Join(args, " "))
		if query == "" {
			return fmt.Errorf("please provide an author to look up (or use --get N)")
		}

		historyRepo := repository.NewHistoryRepository(db)
		history, err := historyRepo.FindAll(ctx)
		if err != nil {
			return fmt.Errorf("load history: %w", err)
		}
		titles := make([]string, 0, len(history))
		for _, h := range history {
			titles = append(titles, h.Title)
		}
		owned := service.NewHistoryOwnedIndex(titles)

		olClient := client.NewOpenLibraryClient(nil)
		lookupService := service.NewLookupService(olClient, owned)

		res, err := lookupService.Lookup(ctx, query)
		if err != nil {
			return err
		}

		cacheRepo := repository.NewCacheRepository(db)
		if err := cacheRepo.AddLookupToCache(ctx, &res.Flat); err != nil {
			fmt.Fprintf(out, "warning: failed to cache lookup: %v\n", err)
		}

		if asJSON {
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(res)
		}

		render.Lookup(out, res, render.Options{Plain: plain, Width: utils.TerminalWidth()})
		return nil
	},
}

func lookupGet(ctx context.Context, db *gorm.DB, out io.Writer, n int) error {
	cacheRepo := repository.NewCacheRepository(db)
	cache, err := cacheRepo.GetLookupCache(ctx)
	if err != nil {
		return fmt.Errorf("no cached lookup found — run 'lamplight lookup <author>' first")
	}

	var books []dao.Book
	if err := json.Unmarshal([]byte(cache.Result), &books); err != nil {
		return fmt.Errorf("error parsing cached lookup: %w", err)
	}
	if n > len(books) {
		return fmt.Errorf("index %d out of range — last lookup had %d books", n, len(books))
	}

	book := books[n-1]
	query := book.Title
	if len(book.Authors) > 0 {
		query = fmt.Sprintf("%s %s", book.Title, book.Authors[0])
	}
	fmt.Fprintf(out, "Searching for: %s\n\n", query)

	return runBookSearch(ctx, db, out, query)
}

func runBookSearch(ctx context.Context, db *gorm.DB, out io.Writer, query string) error {
	repo := repository.NewIndexerRepository(db)
	torznabClient := client.NewTorznabClient(nil)
	torznabBackend := service.NewTorznabBackend(torznabClient)
	searchService := service.NewSearchService(repo, []service.SearchBackend{torznabBackend})

	req := dao.SearchRequest{Query: query, Limit: 15}
	criteria := dao.FilterCriteria{AllowedCategories: constants.BookCategories}

	res, err := searchService.Search(ctx, req, criteria)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}
	if len(res) == 0 {
		fmt.Fprintln(out, "No results.")
		return nil
	}

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

	termW := utils.TerminalWidth()
	titleMax := termW - 68
	if titleMax < 20 {
		titleMax = 20
	}
	if titleMax > 80 {
		titleMax = 80
	}

	_ = utils.PrintOutput(out, string(utils.SEARCH_RESULTS), res, func(d dao.SearchResult) []string {
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

	cache := repository.NewCacheRepository(db)
	if err := cache.AddResultsToCache(ctx, &res); err != nil {
		fmt.Fprintf(out, "warning: failed to cache results: %v\n", err)
	}
	fmt.Fprintln(out, "\nuse 'lamplight download <index>' to grab one.")
	return nil
}

func init() {
	rootCmd.AddCommand(lookupCmd)

	lookupCmd.Flags().Int("get", 0, "search & download book N from your last lookup.")
	lookupCmd.Flags().Bool("plain", false, "plain output — no color or borders.")
	lookupCmd.Flags().Bool("json", false, "output raw JSON.")
}
