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

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/client"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
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
		downRepo := repository.NewDownloaderRepository(db)
		syncSvc := service.NewSyncService(histRepo)

		cfgRepo := repository.NewLibraryConfigRepository(db)
		cfg, err := cfgRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		downloaderClient, clientDetails, err := service.ResolveDownloaderClient(ctx, downRepo)
		if err != nil {
			return fmt.Errorf("connect to downloader: %w", err)
		}

		if err := downloaderClient.Authenticate(ctx); err != nil {
			return fmt.Errorf("can't reach Deluge at %s:%d — %w", clientDetails.Host, clientDetails.Port, err)
		}

		if !watch {
			_, err := renderSync(ctx, out, syncSvc, downloaderClient, cfg.DelugePath, cfg.HostPath, false, 0)
			return err
		}

		// watch mode — trap Ctrl+C for a clean exit
		watchCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
		defer stop()

		prevLines := 0
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// initial draw
		prevLines, err = renderSync(watchCtx, out, syncSvc, downloaderClient, cfg.DelugePath, cfg.HostPath, true, prevLines)
		if err != nil {
			return err
		}

		for {
			select {
			case <-watchCtx.Done():
				fmt.Fprintln(out, "\nstopped.")
				return nil
			case <-ticker.C:
				prevLines, err = renderSync(watchCtx, out, syncSvc, downloaderClient, cfg.DelugePath, cfg.HostPath, true, prevLines)
				if err != nil {
					return err
				}
			}
		}
	},
}

// renderSync runs one sync pass via SyncService and renders it. In watch mode it
// redraws in place using ANSI cursor movement. Returns the number of lines
// written (used by the next redraw to erase them).
func renderSync(
	ctx context.Context,
	out io.Writer,
	syncSvc *service.SyncService,
	dc client.DownloaderClient,
	delugePath, hostPath string,
	watchMode bool,
	prevLines int,
) (int, error) {
	result, err := syncSvc.SyncOnce(ctx, dc, delugePath, hostPath)
	if err != nil {
		return 0, err
	}

	// in watch mode, move cursor back up to overwrite the previous frame
	if watchMode && prevLines > 0 {
		fmt.Fprintf(out, "\033[%dA\033[J", prevLines)
	}

	if len(result.Items) == 0 {
		fmt.Fprint(out, "nothing active to sync.\n")
		return 1, nil
	}

	lines := 0
	if watchMode {
		fmt.Fprintf(out, "  watching %d download(s) — %s — Ctrl+C to stop\n\n",
			len(result.Items), time.Now().Format("15:04:05"))
		lines += 2
	}

	for _, item := range result.Items {
		title := utils.SmartTruncate(item.Entry.Title, 40)
		if item.Err != nil {
			if item.State == "" {
				fmt.Fprintf(out, "  %-40s  error: %v\n", title, item.Err)
			} else {
				fmt.Fprintf(out, "  %-40s  couldn't update: %v\n", title, item.Err)
			}
			lines++
			continue
		}

		if watchMode {
			if item.Done {
				fmt.Fprintf(out, "  ✓ %-38s  done\n", utils.SmartTruncate(item.Entry.Title, 38))
			} else {
				bar := progressBar(item.Progress, 25)
				fmt.Fprintf(out, "  ~ %-38s  %s  %s\n", utils.SmartTruncate(item.Entry.Title, 38), bar, item.State)
			}
		} else {
			if item.Done {
				fmt.Fprintf(out, "  ✓ %-40s  completed\n", title)
			} else {
				fmt.Fprintf(out, "  ~ %-40s  %s (%.0f%%)\n", title, item.State, item.Progress)
			}
		}
		lines++
	}

	if watchMode && result.AllDone && len(result.Items) > 0 {
		fmt.Fprintln(out, "\n  all done.")
		lines += 2
	}

	return lines, nil
}

// progressBar renders a fixed-width ASCII progress bar.
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
