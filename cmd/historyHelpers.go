package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/entity"
	"github.com/intransigent-iconoclast/lamplight-cli/internal/domain/repository"
)

// resolveHistoryEntry finds a history entry by either:
//   - a number → 1-based index into the provided list
//   - a string → case-insensitive fuzzy match on title
//
// if statusFilter is non-empty, index lookups use the filtered list.
func resolveHistoryEntry(
	arg string,
	repo *repository.HistoryRepository,
	entries []entity.DownloadHistory,
) (*entity.DownloadHistory, error) {

	// numeric → index
	if n, err := strconv.Atoi(arg); err == nil {
		if n <= 0 || n > len(entries) {
			return nil, fmt.Errorf("index %d out of range (showing %d entries)", n, len(entries))
		}
		e := entries[n-1]
		return &e, nil
	}

	// string → fuzzy title match
	query := strings.ToLower(arg)
	var matches []entity.DownloadHistory
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Title), query) {
			matches = append(matches, e)
		}
	}

	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no entry found matching %q", arg)
	case 1:
		return &matches[0], nil
	default:
		// multiple matches — show them so the user can be more specific
		msg := fmt.Sprintf("%d entries match %q — be more specific:\n", len(matches), arg)
		for i, m := range matches {
			msg += fmt.Sprintf("  %s  [%s]\n", m.Title, m.Status)
			if i >= 4 {
				msg += fmt.Sprintf("  ... and %d more\n", len(matches)-5)
				break
			}
		}
		return nil, fmt.Errorf(msg)
	}
}
