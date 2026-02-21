package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download <index>",
	Short: "Download a result from the last search.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()

		index, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid index '%s': must be a number", args[0])
		}
		if index <= 0 {
			return fmt.Errorf("index must be greater than 0")
		}

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		cacheRepo := repository.NewCacheRepository(db)

		cache, err := cacheRepo.GetCache(ctx)
		if err != nil {
			return fmt.Errorf("no cached search results found. Run 'lamplight search <query>' first")
		}

		if time.Since(cache.UpdatedAt) > 10*time.Minute {
			fmt.Fprintln(out, "Warning: cached search results are older than 10 minutes.")
		}

		var results []dao.SearchResult
		if err := json.Unmarshal([]byte(cache.Result), &results); err != nil {
			return fmt.Errorf("error parsing cached results: %w", err)
		}

		if len(results) == 0 {
			return fmt.Errorf("cached search results are empty")
		}

		selectedIndex := index - 1
		if selectedIndex >= len(results) {
			return fmt.Errorf(
				"index %d out of range. Last search returned %d results",
				index, len(results),
			)
		}

		selectedResult := results[selectedIndex]

		httpClient := &http.Client{
			Timeout: 20 * time.Second,
		}

		resolved, err := client.Resolve(ctx, httpClient, selectedResult.Link)
		if err != nil {
			return fmt.Errorf("resolve torrent: %w", err)
		}

		downloaderClient, err := createClient(ctx, nil)
		if err != nil {
			return fmt.Errorf("error creating downloader client: %w", err)
		}

		if err := downloaderClient.Add(ctx, resolved); err != nil {
			return fmt.Errorf("failed to add torrent: %w", err)
		}

		fmt.Fprintf(out, "Added: %s\n", selectedResult.Title)
		return nil
	},
}

func createClient(ctx context.Context, clientIndex *int) (client.DownloaderClient, error) {
	db, err := utils.Open("lamplight-cli", false)
	if err != nil {
		return nil, fmt.Errorf("unable to open db: %w", err)
	}

	repo := repository.NewDownloaderRepository(db)

	clientDetails, err := repo.FindHighestPriorityDownloader(ctx)
	if err != nil {
		return nil, err
	}

	switch clientDetails.ClientType {
	case entity.Deluge:
		return client.NewDelugeClient(nil, clientDetails), nil
	default:
		return nil, fmt.Errorf("unsupported downloader type: %s", clientDetails.ClientType)
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}
