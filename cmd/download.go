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
	"gorm.io/gorm"
)

const (
	cacheWarnAfter  = 10 * time.Minute
	cacheStaleAfter = 30 * time.Minute
)

var downloadCmd = &cobra.Command{
	Use:   "download <index>",
	Short: "download a result from your last search.",
	Long: `pick a result by index from your last search and send it to Deluge.

results go stale after 30 minutes — re-run your search if that happens.
use --force to download anyway if you know what you're doing.

  lamplight download 3
  lamplight download 3 --force`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		force, _ := cmd.Flags().GetBool("force")

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
			return fmt.Errorf("no cached results found — run 'lamplight search <query>' first")
		}

		age := time.Since(cache.UpdatedAt)
		if age > cacheStaleAfter && !force {
			return fmt.Errorf(
				"search results are %.0f minutes old — re-run your search or use --force to download anyway",
				age.Minutes(),
			)
		}
		if age > cacheWarnAfter {
			fmt.Fprintf(out, "heads up: these results are %.0f minutes old\n", age.Minutes())
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

		downloaderClient, clientDetails, err := createClient(ctx, db, nil)
		if err != nil {
			return fmt.Errorf("error creating downloader client: %w", err)
		}

		hash, err := downloaderClient.Add(ctx, resolved)
		if err != nil {
			return fmt.Errorf("failed to add torrent: %w", err)
		}

		// Record to history — non-fatal if this fails
		historyRepo := repository.NewHistoryRepository(db)
		var sizeBytes int64
		if selectedResult.SizeBytes != nil {
			sizeBytes = *selectedResult.SizeBytes
		}
		entry := entity.DownloadHistory{
			Title:          selectedResult.Title,
			Link:           selectedResult.Link,
			IndexerName:    selectedResult.IndexerName,
			DownloaderName: clientDetails.Name,
			SizeBytes:      sizeBytes,
			Status:         entity.StatusSnatched,
			TorrentHash:    hash,
		}
		if err := historyRepo.Save(ctx, &entry); err != nil {
			fmt.Fprintf(out, "warning: failed to record download history: %v\n", err)
		}

		fmt.Fprintf(out, "Added: %s\n", selectedResult.Title)
		return nil
	},
}

func createClient(ctx context.Context, db *gorm.DB, clientIndex *int) (client.DownloaderClient, *entity.Downloader, error) {
	repo := repository.NewDownloaderRepository(db)

	clientDetails, err := repo.FindHighestPriorityDownloader(ctx)
	if err != nil {
		return nil, nil, err
	}

	switch clientDetails.ClientType {
	case entity.Deluge:
		return client.NewDelugeClient(nil, clientDetails), clientDetails, nil
	default:
		return nil, nil, fmt.Errorf("unsupported downloader type: %s", clientDetails.ClientType)
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolP("force", "f", false, "download even if the search results are stale")
}
