package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/internal/util"
	"github.com/spf13/cobra"
)

// bookExtensions are the file types we'll process.
var bookExtensions = map[string]bool{
	".epub": true,
	".pdf":  true,
	".mobi": true,
	".azw":  true,
	".azw3": true,
	".mp3":  true,
	".m4b":  true,
	".m4a":  true,
	".cbz":  true,
	".cbr":  true,
}

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

		libraryPath := expandHome(cfg.LibraryPath)

		// --- no path: process completed history entries ---
		if len(args) == 0 {
			histRepo := repository.NewHistoryRepository(db)
			completed, err := histRepo.FindCompleted(ctx)
			if err != nil {
				return fmt.Errorf("load completed downloads: %w", err)
			}

			if len(completed) == 0 {
				fmt.Fprintln(out, "No completed downloads to organize. Run 'lamplight history sync' first.")
				return nil
			}

			for _, entry := range completed {
				dest, placed, organizeErr := organizeFile(entry.FilePath, libraryPath, cfg.Template, dryRun)
				if organizeErr != nil {
					fmt.Fprintf(out, "  skip  %s — %v\n", utils.SmartTruncate(entry.Title, 50), organizeErr)
					continue
				}

				if !dryRun {
					_ = histRepo.UpdateStatusAndPath(ctx, entry.ID, entity.StatusCompleted, dest)
				}

				if placed == "library" {
					fmt.Fprintf(out, "  ok    %s\n        → library/%s\n", utils.SmartTruncate(entry.Title, 50), dest)
				} else {
					fmt.Fprintf(out, "  ok    %s\n        → uncategorized/%s\n", utils.SmartTruncate(entry.Title, 50), filepath.Base(dest))
				}
			}

			return nil
		}

		// --- path provided: one-off manual organize ---
		inputPath := args[0]
		info, err := os.Stat(inputPath)
		if err != nil {
			return fmt.Errorf("can't access '%s': %w", inputPath, err)
		}

		var files []string
		if info.IsDir() {
			return fmt.Errorf("organizing a raw directory is no longer supported — use 'lamplight history sync' then 'lamplight organize'")
		}

		if !bookExtensions[strings.ToLower(filepath.Ext(inputPath))] {
			return fmt.Errorf("'%s' doesn't look like a book file", filepath.Base(inputPath))
		}
		files = []string{inputPath}

		for _, file := range files {
			dest, placed, organizeErr := organizeFile(file, libraryPath, cfg.Template, dryRun)
			if organizeErr != nil {
				fmt.Fprintf(out, "  skip  %s — %v\n", filepath.Base(file), organizeErr)
				continue
			}
			if placed == "library" {
				fmt.Fprintf(out, "  ok    %s\n        → library/%s\n", filepath.Base(file), dest)
			} else {
				fmt.Fprintf(out, "  ok    %s\n        → uncategorized/%s\n", filepath.Base(file), filepath.Base(dest))
			}
		}

		return nil
	},
}

// organizeFile moves a single file to the right place in the library.
// Returns (relative-dest-path, "library"|"uncategorized", error).
func organizeFile(src, libraryRoot, tmpl string, dryRun bool) (string, string, error) {
	meta, err := utils.ReadMetadata(src)
	if err != nil {
		_ = err // non-fatal — fall back to uncategorized
	}

	ext := strings.ToLower(filepath.Ext(src))

	var destDir, relPath string
	var placed string

	if utils.IsComplete(meta) {
		relPath = utils.ApplyTemplate(tmpl, meta) + ext
		destDir = filepath.Join(libraryRoot, "library", filepath.Dir(relPath))
		placed = "library"
	} else {
		destDir = filepath.Join(libraryRoot, "uncategorized")
		relPath = filepath.Base(src)
		placed = "uncategorized"
	}

	destFile := filepath.Join(destDir, filepath.Base(relPath))
	destFile = resolveConflict(destFile)

	if dryRun {
		return relPath, placed, nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create directory: %w", err)
	}

	if err := moveFile(src, destFile); err != nil {
		return "", "", fmt.Errorf("move file: %w", err)
	}

	return relPath, placed, nil
}

// resolveConflict appends _2, _3, … to the stem if the path already exists.
func resolveConflict(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return path
}

// moveFile tries os.Rename first (fast, same-device), falls back to copy+delete.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

// collectBookFiles walks a directory and returns all book file paths.
func collectBookFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && bookExtensions[strings.ToLower(filepath.Ext(path))] {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// expandHome replaces a leading ~ with the actual home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

func init() {
	rootCmd.AddCommand(organizeCmd)
	organizeCmd.Flags().Bool("dry-run", false, "Show what would happen without moving anything.")
}
