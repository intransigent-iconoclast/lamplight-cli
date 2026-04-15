package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/client"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

var syncHistoryCmd = &cobra.Command{
	Use:   "sync",
	Short: "check Deluge for status updates on active downloads.",
	Long: `polls Deluge for the current state of all active downloads and updates history.

  lamplight history sync

use --watch / -w to get a live view that refreshes every second:

  lamplight history sync -w

press Ctrl+C to exit watch mode.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		watch, _ := cmd.Flags().GetBool("watch")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		histRepo := repository.NewHistoryRepository(db)

		cfgRepo := repository.NewLibraryConfigRepository(db)
		cfg, err := cfgRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		downloaderClient, clientDetails, err := createClient(ctx, db, nil)
		if err != nil {
			return fmt.Errorf("connect to downloader: %w", err)
		}

		if err := downloaderClient.Authenticate(ctx); err != nil {
			return fmt.Errorf("can't reach Deluge at %s:%d — %w", clientDetails.Host, clientDetails.Port, err)
		}

		if !watch {
			_, err := doSync(ctx, out, histRepo, downloaderClient, cfg.DelugePath, cfg.HostPath, false, 0)
			return err
		}

		// watch mode — trap Ctrl+C for a clean exit
		watchCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		prevLines := 0
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// initial draw
		prevLines, err = doSync(watchCtx, out, histRepo, downloaderClient, cfg.DelugePath, cfg.HostPath, true, prevLines)
		if err != nil {
			return err
		}

		for {
			select {
			case <-watchCtx.Done():
				fmt.Fprintln(out, "\nstopped.")
				return nil
			case <-ticker.C:
				prevLines, err = doSync(watchCtx, out, histRepo, downloaderClient, cfg.DelugePath, cfg.HostPath, true, prevLines)
				if err != nil {
					return err
				}
			}
		}
	},
}

// doSync polls Deluge for all active entries and updates history.
// In watch mode it redraws in place using ANSI cursor movement.
// Returns the number of lines written (used by the next redraw to erase them).
func doSync(
	ctx context.Context,
	out io.Writer,
	histRepo *repository.HistoryRepository,
	dc client.DownloaderClient,
	delugePath, hostPath string,
	watchMode bool,
	prevLines int,
) (int, error) {
	active, err := histRepo.FindActive(ctx)
	if err != nil {
		return 0, fmt.Errorf("load active downloads: %w", err)
	}

	// in watch mode, move cursor back up to overwrite the previous frame
	if watchMode && prevLines > 0 {
		fmt.Fprintf(out, "\033[%dA\033[J", prevLines)
	}

	if len(active) == 0 {
		msg := "nothing active to sync.\n"
		fmt.Fprint(out, msg)
		return 1, nil
	}

	lines := 0

	if watchMode {
		header := fmt.Sprintf("  watching %d download(s) — %s — Ctrl+C to stop\n\n",
			len(active), time.Now().Format("15:04:05"))
		fmt.Fprint(out, header)
		lines += 2
	}

	allDone := true
	for _, entry := range active {
		status, err := dc.GetTorrentStatus(ctx, entry.TorrentHash)
		if err != nil {
			line := fmt.Sprintf("  %-40s  error: %v\n", utils.SmartTruncate(entry.Title, 40), err)
			fmt.Fprint(out, line)
			lines++
			allDone = false
			continue
		}

		filePath := translatePath(status.FilePath, delugePath, hostPath)
		newStatus, done := delugeStateToStatus(status.State)

		if !done {
			allDone = false
		}

		if err := histRepo.UpdateStatusAndPath(ctx, entry.ID, newStatus, filePath); err != nil {
			line := fmt.Sprintf("  %-40s  couldn't update: %v\n", utils.SmartTruncate(entry.Title, 40), err)
			fmt.Fprint(out, line)
			lines++
			continue
		}

		if watchMode {
			if done {
				line := fmt.Sprintf("  ✓ %-38s  done\n", utils.SmartTruncate(entry.Title, 38))
				fmt.Fprint(out, line)
			} else {
				bar := progressBar(status.Progress, 25)
				line := fmt.Sprintf("  ~ %-38s  %s  %s\n",
					utils.SmartTruncate(entry.Title, 38), bar, status.State)
				fmt.Fprint(out, line)
			}
		} else {
			if done {
				fmt.Fprintf(out, "  ✓ %-40s  completed\n", utils.SmartTruncate(entry.Title, 40))
			} else {
				fmt.Fprintf(out, "  ~ %-40s  %s (%.0f%%)\n",
					utils.SmartTruncate(entry.Title, 40), status.State, status.Progress)
			}
		}
		lines++
	}

	if watchMode && allDone && len(active) > 0 {
		fmt.Fprintln(out, "\n  all done.")
		lines += 2
	}

	return lines, nil
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
func translatePath(path, delugePath, hostPath string) string {
	if delugePath == "" || hostPath == "" || !strings.HasPrefix(path, delugePath) {
		return path
	}
	return hostPath + path[len(delugePath):]
}

// progressBar renders a fixed-width ASCII progress bar.
// e.g. [████████████░░░░░░░░░░░░░] 60.0%
func progressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}
	filled := int(progress / 100.0 * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return fmt.Sprintf("[%s] %5.1f%%", bar, progress)
}

func init() {
	historyCmd.AddCommand(syncHistoryCmd)
	syncHistoryCmd.Flags().BoolP("watch", "w", false, "live progress view, refreshes every 3 seconds")
}
