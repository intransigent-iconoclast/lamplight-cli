package service

import (
	"sort"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
)

// FilterByFormat narrows results to those matching one of the comma-separated
// format/type aliases (e.g. "epub,audiobook"). "all" (anywhere in the list)
// disables filtering. An empty typeFilter also disables filtering. Comparison
// is case-insensitive.
func FilterByFormat(results []dao.SearchResult, typeFilter string) []dao.SearchResult {
	typeFilter = strings.TrimSpace(typeFilter)
	if typeFilter == "" {
		return results
	}

	types := strings.Split(strings.ToLower(typeFilter), ",")
	for _, t := range types {
		if strings.TrimSpace(t) == "all" {
			return results
		}
	}

	var filtered []dao.SearchResult
	for _, r := range results {
		for _, t := range types {
			if matchesTypeFilter(r.Format, strings.TrimSpace(t)) {
				filtered = append(filtered, r)
				break
			}
		}
	}
	return filtered
}

// matchesTypeFilter maps content-type aliases to detected format values.
// "book"/"ebook" match any prose format (epub/pdf/mobi); other filters
// (audiobook, comic, epub, pdf, mobi, unknown) match themselves exactly.
func matchesTypeFilter(format, filter string) bool {
	switch filter {
	case "book", "ebook":
		return format == "epub" || format == "pdf" || format == "mobi"
	default:
		return format == filter
	}
}

// SortKey identifies how SortResults should order results.
type SortKey string

const (
	SortBySeeders  SortKey = "seeders"
	SortByLeechers SortKey = "leechers"
	SortBySize     SortKey = "size"
	SortByTitle    SortKey = "title"
)

// SortResults stably sorts results in place by the given key. An unrecognized
// key is a no-op, matching CLI behavior.
func SortResults(results []dao.SearchResult, sortBy SortKey) {
	switch sortBy {
	case SortBySeeders:
		sort.SliceStable(results, func(i, j int) bool {
			return intOrZero(results[i].Seeders) > intOrZero(results[j].Seeders)
		})
	case SortByLeechers:
		sort.SliceStable(results, func(i, j int) bool {
			return intOrZero(results[i].Leechers) > intOrZero(results[j].Leechers)
		})
	case SortBySize:
		sort.SliceStable(results, func(i, j int) bool {
			return int64OrZero(results[i].SizeBytes) > int64OrZero(results[j].SizeBytes)
		})
	case SortByTitle:
		sort.SliceStable(results, func(i, j int) bool {
			return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
		})
	}
}

func intOrZero(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func int64OrZero(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
