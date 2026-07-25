package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
)

// reference https://openlibrary.org/swagger/docs
type MetadataProvider interface {
	Lookup(ctx context.Context, query string) (*dao.Author, error)
	SearchBooks(ctx context.Context, query string) ([]dao.Book, error)
}

type OwnedIndex interface {
	Owns(title string) bool
}

type Series struct {
	Name  string
	Books []dao.Book
}

// Flat is every book in display order (series first, then standalone) so a
// 1-based index into it matches the rendered list and `lookup --get N`.
type LookupResult struct {
	Author     dao.Author
	Series     []Series
	Standalone []dao.Book
	Flat       []dao.Book
}

type LookupService struct {
	provider MetadataProvider
	owned    OwnedIndex
}

func NewLookupService(provider MetadataProvider, owned OwnedIndex) *LookupService {
	return &LookupService{provider: provider, owned: owned}
}

func (s *LookupService) Lookup(ctx context.Context, query string) (*LookupResult, error) {
	author, err := s.provider.Lookup(ctx, query)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, fmt.Errorf("no author found matching %q", query)
	}

	for i := range author.Books {
		if s.owned != nil {
			author.Books[i].Owned = s.owned.Owns(author.Books[i].Title)
		}
	}

	result := &LookupResult{Author: *author}
	groupBooks(result, author.Books)
	return result, nil
}

// SearchBooks runs a general catalog search (title/author/keyword) and marks
// each result owned against download history. Results stay in the provider's
// relevance order and are flat — no series grouping, since a query can span
// many authors and works.
func (s *LookupService) SearchBooks(ctx context.Context, query string) ([]dao.Book, error) {
	books, err := s.provider.SearchBooks(ctx, query)
	if err != nil {
		return nil, err
	}
	for i := range books {
		if s.owned != nil {
			books[i].Owned = s.owned.Owns(books[i].Title)
		}
	}
	return books, nil
}

func groupBooks(result *LookupResult, books []dao.Book) {
	bySeries := map[string][]dao.Book{}
	var standalone []dao.Book
	for _, b := range books {
		if strings.TrimSpace(b.SeriesName) == "" {
			standalone = append(standalone, b)
			continue
		}
		bySeries[b.SeriesName] = append(bySeries[b.SeriesName], b)
	}

	names := make([]string, 0, len(bySeries))
	for name := range bySeries {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		group := bySeries[name]
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].SeriesPos != group[j].SeriesPos {
				return group[i].SeriesPos < group[j].SeriesPos
			}
			return group[i].Year < group[j].Year
		})
		result.Series = append(result.Series, Series{Name: name, Books: group})
		result.Flat = append(result.Flat, group...)
	}

	result.Standalone = standalone
	result.Flat = append(result.Flat, standalone...)
}

// Marks a book owned when its title appears within any download-history title,
// which often carry format/author junk around it.
type HistoryOwnedIndex struct {
	normalizedTitles []string
}

func NewHistoryOwnedIndex(historyTitles []string) *HistoryOwnedIndex {
	normalized := make([]string, 0, len(historyTitles))
	for _, t := range historyTitles {
		normalized = append(normalized, normalizeTitle(t))
	}
	return &HistoryOwnedIndex{normalizedTitles: normalized}
}

func (h *HistoryOwnedIndex) Owns(title string) bool {
	needle := normalizeTitle(title)
	if needle == "" {
		return false
	}
	for _, hay := range h.normalizedTitles {
		if strings.Contains(hay, needle) {
			return true
		}
	}
	return false
}

func normalizeTitle(s string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSpace = false
		case r == ' ':
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}
