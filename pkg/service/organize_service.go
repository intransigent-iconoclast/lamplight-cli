package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/pkg/domain/repository"
	utils "github.com/intransigent-iconoclast/lamplight-cli/pkg/util"
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

// audioExtensions identifies audiobook file types for path routing.
var audioExtensions = map[string]bool{
	".mp3": true,
	".m4b": true,
	".m4a": true,
}

// comicExtensions identifies comic/manga file types for path routing.
var comicExtensions = map[string]bool{
	".cbz": true,
	".cbr": true,
}

// IsBookFile reports whether a path has a recognized book/media extension.
func IsBookFile(path string) bool {
	return bookExtensions[strings.ToLower(filepath.Ext(path))]
}

// OrganizeService moves completed downloads into the configured library. It holds
// no presentation logic — callers render OrganizeReport / OrganizeItem as they like.
type OrganizeService struct {
	histRepo *repository.HistoryRepository
}

func NewOrganizeService(histRepo *repository.HistoryRepository) *OrganizeService {
	return &OrganizeService{histRepo: histRepo}
}

// OrganizeItem is the outcome of organizing one download (or manual path).
type OrganizeItem struct {
	Title   string
	Placed  string // "library" | "uncategorized"
	Dest    string // relative destination to display
	Already bool   // already inside a library root (no-op)
	Err     error
}

// OrganizeReport summarizes an OrganizeCompleted run.
type OrganizeReport struct {
	LibraryPath   string
	AudiobookPath string
	ComicsPath    string
	Items         []OrganizeItem
	Moved         int
	Skipped       int
	Already       int
}

// OrganizeCompleted organizes every completed history entry into the library.
func (s *OrganizeService) OrganizeCompleted(ctx context.Context, cfg *entity.LibraryConfig, dryRun bool) (*OrganizeReport, error) {
	libraryPath := expandHome(cfg.LibraryPath)
	audiobookPath := expandHome(cfg.AudiobookPath)
	comicsPath := expandHome(cfg.ComicsPath)

	report := &OrganizeReport{
		LibraryPath:   libraryPath,
		AudiobookPath: audiobookPath,
		ComicsPath:    comicsPath,
	}

	completed, err := s.histRepo.FindCompleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("load completed downloads: %w", err)
	}

	for _, entry := range completed {
		filePath := entry.FilePath
		if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) && cfg.DelugePath != "" {
			filePath = utils.TranslatePath(filePath, cfg.DelugePath, cfg.HostPath)
		}

		// already inside the library — nothing to do
		if strings.HasPrefix(filePath, libraryPath) ||
			(audiobookPath != "" && strings.HasPrefix(filePath, audiobookPath)) ||
			(comicsPath != "" && strings.HasPrefix(filePath, comicsPath)) {
			report.Already++
			report.Items = append(report.Items, OrganizeItem{Title: entry.Title, Already: true})
			continue
		}

		dest, placed, organizeErr := organizeEntry(filePath, libraryPath, audiobookPath, comicsPath, cfg.Template, dryRun)
		if organizeErr != nil {
			report.Skipped++
			report.Items = append(report.Items, OrganizeItem{Title: entry.Title, Err: organizeErr})
			continue
		}

		if !dryRun {
			_ = s.histRepo.UpdateStatusAndPath(ctx, entry.ID, entity.StatusCompleted, dest)
		}

		report.Moved++
		report.Items = append(report.Items, OrganizeItem{
			Title:  entry.Title,
			Placed: placed,
			Dest:   displayDest(dest, placed),
		})
	}

	return report, nil
}

// OrganizePath organizes a single manual file or directory.
func (s *OrganizeService) OrganizePath(inputPath string, cfg *entity.LibraryConfig, dryRun bool) (*OrganizeItem, error) {
	libraryPath := expandHome(cfg.LibraryPath)
	audiobookPath := expandHome(cfg.AudiobookPath)
	comicsPath := expandHome(cfg.ComicsPath)

	info, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("can't access '%s': %w", inputPath, err)
	}
	if !info.IsDir() && !IsBookFile(inputPath) {
		return nil, fmt.Errorf("'%s' doesn't look like a book file", filepath.Base(inputPath))
	}

	label := filepath.Base(inputPath)
	dest, placed, organizeErr := organizeEntry(inputPath, libraryPath, audiobookPath, comicsPath, cfg.Template, dryRun)
	if organizeErr != nil {
		return &OrganizeItem{Title: label, Err: organizeErr}, nil
	}
	return &OrganizeItem{Title: label, Placed: placed, Dest: displayDest(dest, placed)}, nil
}

// displayDest formats the relative destination for presentation.
func displayDest(dest, placed string) string {
	if placed == "library" {
		return dest
	}
	return "uncategorized/" + filepath.Base(dest)
}

// --- file engine (pure, no presentation) ---

// pickRoot returns the right destination root for a file based on its type.
func pickRoot(src, libraryRoot, audiobookRoot, comicsRoot string) string {
	ext := strings.ToLower(filepath.Ext(src))
	if audiobookRoot != "" && audioExtensions[ext] {
		return audiobookRoot
	}
	if comicsRoot != "" && comicExtensions[ext] {
		return comicsRoot
	}
	return libraryRoot
}

// pickRootForDir returns the right root for a directory of files.
func pickRootForDir(files []string, libraryRoot, audiobookRoot, comicsRoot string) string {
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		if audiobookRoot != "" && audioExtensions[ext] {
			return audiobookRoot
		}
		if comicsRoot != "" && comicExtensions[ext] {
			return comicsRoot
		}
	}
	return libraryRoot
}

// organizeFile moves a single file to the right place in the library.
// Returns (relative-dest-path, "library"|"uncategorized", error).
func organizeFile(src, libraryRoot, audiobookRoot, comicsRoot, tmpl string, dryRun bool) (string, string, error) {
	meta, err := utils.ReadMetadata(src)
	if err != nil {
		_ = err // non-fatal — fall back to uncategorized
	}

	ext := strings.ToLower(filepath.Ext(src))
	root := pickRoot(src, libraryRoot, audiobookRoot, comicsRoot)

	var destDir, relPath string
	var placed string

	if utils.IsComplete(meta) {
		relPath = utils.ApplyTemplate(tmpl, meta) + ext
		destDir = filepath.Join(root, filepath.Dir(relPath))
		placed = "library"
	} else {
		destDir = filepath.Join(root, "uncategorized")
		relPath = filepath.Base(src)
		placed = "uncategorized"
	}

	destFile := filepath.Join(destDir, filepath.Base(relPath))
	destFile = resolveConflict(destFile)

	// re-derive relPath from the actual destination after conflict resolution
	if placed == "library" {
		relPath = strings.TrimPrefix(destFile, root+string(filepath.Separator))
	} else {
		relPath = filepath.Base(destFile)
	}

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

// organizeEntry dispatches to organizeDir or organizeFile depending on what src is.
func organizeEntry(src, libraryRoot, audiobookRoot, comicsRoot, tmpl string, dryRun bool) (string, string, error) {
	info, err := os.Stat(src)
	if err != nil {
		return "", "", fmt.Errorf("can't access '%s': %w", filepath.Base(src), err)
	}
	if info.IsDir() {
		return organizeDir(src, libraryRoot, audiobookRoot, comicsRoot, tmpl, dryRun)
	}
	return organizeFile(src, libraryRoot, audiobookRoot, comicsRoot, tmpl, dryRun)
}

// organizeDir handles a folder of book files (audiobook chapters, comic series).
// Single-file folders are unwrapped; multi-file folders are moved as a group.
func organizeDir(src, libraryRoot, audiobookRoot, comicsRoot, tmpl string, dryRun bool) (string, string, error) {
	files, err := collectBookFiles(src)
	if err != nil {
		return "", "", fmt.Errorf("scan directory: %w", err)
	}
	if len(files) == 0 {
		return "", "", fmt.Errorf("no book files found inside")
	}

	if len(files) == 1 {
		return organizeFile(files[0], libraryRoot, audiobookRoot, comicsRoot, tmpl, dryRun)
	}

	meta := bestMetadata(files)
	root := pickRootForDir(files, libraryRoot, audiobookRoot, comicsRoot)

	var destDir, relPath, placed string
	if utils.IsComplete(meta) {
		relPath = utils.ApplyTemplate(tmpl, meta)
		destDir = filepath.Join(root, relPath)
		placed = "library"
	} else {
		relPath = filepath.Base(src)
		destDir = filepath.Join(root, "uncategorized", relPath)
		placed = "uncategorized"
	}

	destDir = resolveConflictDir(destDir)

	if dryRun {
		return relPath, placed, nil
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", fmt.Errorf("create directory: %w", err)
	}

	for _, f := range files {
		dst := filepath.Join(destDir, filepath.Base(f))
		if err := moveFile(f, dst); err != nil {
			return "", "", fmt.Errorf("move %s: %w", filepath.Base(f), err)
		}
	}

	// clean up the source directory (RemoveAll handles nested subdirs)
	_ = os.RemoveAll(src)

	return relPath, placed, nil
}

// bestMetadata picks the most useful metadata from a list of files.
func bestMetadata(files []string) *utils.BookMetadata {
	audioExts := map[string]bool{".mp3": true, ".m4b": true, ".m4a": true}

	candidates := files
	var audioFiles []string
	for _, f := range files {
		if audioExts[strings.ToLower(filepath.Ext(f))] {
			audioFiles = append(audioFiles, f)
		}
	}
	if len(audioFiles) > 0 {
		candidates = audioFiles
	}

	for _, f := range candidates {
		meta, err := utils.ReadMetadata(f)
		if err == nil && utils.IsComplete(meta) {
			return meta
		}
	}

	meta, _ := utils.ReadMetadata(candidates[0])
	return meta
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

// resolveConflictDir appends _2, _3, … to the directory name if it already exists.
func resolveConflictDir(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	for i := 2; i < 100; i++ {
		candidate := fmt.Sprintf("%s_%d", path, i)
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
