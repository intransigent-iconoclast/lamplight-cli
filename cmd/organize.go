package cmd

import (
	"fmt"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/service"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
	"github.com/spf13/cobra"
)

var organizeCmd = &cobra.Command{
	Use:   "organize [path]",
	Short: "Move completed downloads into your library.",
	Long: `Without a path, organizes all completed lamplight downloads.
Run 'lamplight history sync' first to update download statuses.

  lamplight history sync
  lamplight organize

You can also point it at a specific file manually:

  lamplight organize ~/Downloads/some-book.epub

Files with enough metadata (author + title) go into:
  <library-path>/library/<template>.<ext>

Everything else ends up in:
  <library-path>/uncategorized/<filename>`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		out := cmd.OutOrStdout()
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		db, err := utils.Open("lamplight-cli", false)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		cfgRepo := repository.NewLibraryConfigRepository(db)
		cfg, err := cfgRepo.Get(ctx)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		organizeSvc := service.NewOrganizeService(repository.NewHistoryRepository(db))

		// --- path provided: one-off manual organize ---
		if len(args) > 0 {
			item, err := organizeSvc.OrganizePath(args[0], cfg, dryRun)
			if err != nil {
				return err
			}
			printOrganizeItem(cmd, *item)
			return nil
		}

		// --- no path: process completed history entries ---
		report, err := organizeSvc.OrganizeCompleted(ctx, cfg, dryRun)
		if err != nil {
			return err
		}

		if report.Moved == 0 && report.Skipped == 0 && report.Already == 0 && len(report.Items) == 0 {
			fmt.Fprintln(out, "Nothing to organize — run 'lamplight history sync' first if you're expecting something.")
			return nil
		}

		if dryRun {
			fmt.Fprintln(out, "Dry run — nothing will be moved.")
		} else {
			fmt.Fprintf(out, "Library → %s\n", report.LibraryPath)
			if report.AudiobookPath != "" {
				fmt.Fprintf(out, "Audiobooks → %s\n", report.AudiobookPath)
			}
			if report.ComicsPath != "" {
				fmt.Fprintf(out, "Comics → %s\n", report.ComicsPath)
			}
			fmt.Fprintln(out)
		}

		for _, item := range report.Items {
			if item.Already {
				continue // already organized — don't clutter the output
			}
			printOrganizeItem(cmd, item)
		}

		fmt.Fprintln(out)
		if report.Moved == 0 && report.Skipped == 0 {
			fmt.Fprintf(out, "All %d downloads already organized.\n", report.Already)
		} else {
			if report.Moved > 0 {
				fmt.Fprintf(out, "Moved %d", report.Moved)
			}
			if report.Skipped > 0 {
				fmt.Fprintf(out, "  Skipped %d", report.Skipped)
			}
			if report.Already > 0 {
				fmt.Fprintf(out, "  Already organized %d", report.Already)
			}
			fmt.Fprintln(out)
		}

		return nil
	},
}

// printOrganizeItem renders a single organize outcome.
func printOrganizeItem(cmd *cobra.Command, item service.OrganizeItem) {
	out := cmd.OutOrStdout()
	title := utils.SmartTruncate(item.Title, 50)
	switch {
	case item.Err != nil:
		fmt.Fprintf(out, "  ✗  %s\n     %v\n", title, item.Err)
	case item.Placed == "library":
		fmt.Fprintf(out, "  ✓  %s\n     → %s\n", title, item.Dest)
	default:
		fmt.Fprintf(out, "  →  %s\n     → %s\n", title, item.Dest)
	}
}

func init() {
	rootCmd.AddCommand(organizeCmd)
	organizeCmd.Flags().Bool("dry-run", false, "Show what would happen without moving anything.")
}
