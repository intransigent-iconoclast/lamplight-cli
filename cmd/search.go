/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/constants"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "A brief description of your command",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// verify the input flags ... i'll double check if there are rules here later like can author only be used with book query type... not sure
		query, _ := cmd.Flags().GetString("query")
		if strings.TrimSpace(query) == "" {
			return fmt.Errorf("Error: Please provide a value for --query")
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
		// res, err := repo.FindByEnabled(ctx)
		// fmt.Println("rando:", res)

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

			// this is what SearchService.resolveIndexers will key on
			searchRequest.IndexerName = indexers[indexer].Name
		}

		// will need to add a big list here before long of different backends
		torznabClient := client.NewTorznabClient(nil) // default client is used in this case ... probably just remote the parameter unless i need to fiddle with it later idk ill leave it
		torznabBackend := service.NewTorznabBackend(torznabClient)

		searchService := service.NewSearchService(repo, []service.SearchBackend{torznabBackend})

		// construct filter criterion
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

		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TITLE\tINDEXER\tSIZE_BYTES\tSEEDERS\tLEECHERS")

		for _, r := range res {
			var sizeStr, seedStr, leechStr string
			if r.SizeBytes != nil {
				sizeStr = fmt.Sprintf("%d", *r.SizeBytes)
			}
			if r.Seeders != nil {
				seedStr = fmt.Sprintf("%d", *r.Seeders)
			}
			if r.Leechers != nil {
				leechStr = fmt.Sprintf("%d", *r.Leechers)
			}

			fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\n",
				strings.TrimSpace(r.Title),
				strings.TrimSpace(r.IndexerName),
				sizeStr,
				seedStr,
				leechStr,
			)
		}

		if err := w.Flush(); err != nil {
			return fmt.Errorf("flush writer: %w", err)
		}

		return nil

	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringP("query", "q", "", "The book you are searching for. E.g., \"The Lord of the Rings\"")
	searchCmd.Flags().IntP("indexer", "i", -1, "Search using a specific indexer. Use the Index number acquired from the list command.")
	searchCmd.Flags().IntP("limit", "l", 15, "The maximum number of items you'd like back.")
	searchCmd.Flags().BoolP("books", "b", true, "Filters by books - true by default .. since this is for books right now")

	// searchCmd.MarkFlagRequired("query")
}
