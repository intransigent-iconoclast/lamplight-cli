package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
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

		downloadSvc := service.NewDownloadService(
			repository.NewHistoryRepository(db),
			repository.NewDownloaderRepository(db),
			&http.Client{Timeout: 20 * time.Second},
		)

		res, err := downloadSvc.Dispatch(ctx, selectedResult, force)
		if err != nil {
			if errors.Is(err, service.ErrAlreadyInHistory) {
				return fmt.Errorf("'%s' is already in your history — use --force to download it again", selectedResult.Title)
			}
			return err
		}

		if res.HistoryWarning != nil {
			fmt.Fprintf(out, "warning: failed to record download history: %v\n", res.HistoryWarning)
		}

		fmt.Fprintf(out, "Added: %s\n", res.Title)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().BoolP("force", "f", false, "download even if the search results are stale")
}
