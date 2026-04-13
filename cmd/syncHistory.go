package cmd

import (
	"fmt"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var syncHistoryCmd = &cobra.Command{
	Use:   "sync",
	Short: "check Deluge for status updates on active downloads.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		histRepo := repository.NewHistoryRepository(db)
		active, err := histRepo.FindActive(ctx)
		if err != nil {
			return fmt.Errorf("load active downloads: %w", err)
		}

		if len(active) == 0 {
			fmt.Fprintln(out, "Nothing active to sync.")
			return nil
		}

		// load path translation config (handles docker path mismatch)
		cfgRepo := repository.NewLibraryConfigRepository(db)
		cfg, err := cfgRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		downloaderClient, _, err := createClient(ctx, db, nil)
		if err != nil {
			return fmt.Errorf("connect to downloader: %w", err)
		}

		for _, entry := range active {
			status, err := downloaderClient.GetTorrentStatus(ctx, entry.TorrentHash)
			if err != nil {
				fmt.Fprintf(out, "  %-40s  error: %v\n", utils.SmartTruncate(entry.Title, 40), err)
				continue
			}

			// translate container path → host path if configured
			filePath := translatePath(status.FilePath, cfg.DelugePath, cfg.HostPath)

			newStatus, done := delugeStateToStatus(status.State)

			if err := histRepo.UpdateStatusAndPath(ctx, entry.ID, newStatus, filePath); err != nil {
				fmt.Fprintf(out, "  %-40s  couldn't update: %v\n", utils.SmartTruncate(entry.Title, 40), err)
				continue
			}

			if done {
				fmt.Fprintf(out, "  ✓ %-40s  completed → %s\n", utils.SmartTruncate(entry.Title, 40), filePath)
			} else {
				fmt.Fprintf(out, "  ~ %-40s  %s (%.0f%%)\n", utils.SmartTruncate(entry.Title, 40), status.State, status.Progress)
			}
		}

		return nil
	},
}

// delugeStateToStatus maps a Deluge state string to our status enum.
// Returns (status, isComplete).
func delugeStateToStatus(state string) (entity.DownloadStatus, bool) {
	switch state {
	case "Seeding": // 100% downloaded, now seeding
		return entity.StatusCompleted, true
	case "Error":
		return entity.StatusFailed, false
	case "Downloading", "Checking", "Moving":
		return entity.StatusDownloading, false
	default: // Queued, Paused, etc.
		return entity.StatusSnatched, false
	}
}

// translatePath replaces a container path prefix with the real host path.
// e.g. /data/incomplete/foo → /opt/docker/data/.../downloads/incomplete/foo
func translatePath(path, delugePath, hostPath string) string {
	if delugePath == "" || hostPath == "" || !strings.HasPrefix(path, delugePath) {
		return path
	}
	return hostPath + path[len(delugePath):]
}

func init() {
	historyCmd.AddCommand(syncHistoryCmd)
}
