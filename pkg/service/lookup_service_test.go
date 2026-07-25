package service

import (
	"context"
	"testing"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubProvider struct {
	author *dao.Author
	books  []dao.Book
}

func (s stubProvider) Lookup(_ context.Context, _ string) (*dao.Author, error) {
	return s.author, nil
}

func (s stubProvider) SearchBooks(_ context.Context, _ string) ([]dao.Book, error) {
	return s.books, nil
}

func TestLookupGroupsSeriesAndMarksOwned(t *testing.T) {
	author := &dao.Author{
		Name: "Becky Chambers",
		Books: []dao.Book{
			{Title: "A Psalm for the Wild-Built", SeriesName: "Monk & Robot", SeriesPos: 1, Year: 2021},
			{Title: "Record of a Spaceborn Few", SeriesName: "Wayfarers", SeriesPos: 3, Year: 2018},
			{Title: "The Long Way to a Small, Angry Planet", SeriesName: "Wayfarers", SeriesPos: 1, Year: 2014},
			{Title: "To Be Taught, If Fortunate", Year: 2019}, // standalone
		},
	}

	owned := NewHistoryOwnedIndex([]string{"Becky Chambers - The Long Way to a Small, Angry Planet (epub)"})
	svc := NewLookupService(stubProvider{author: author}, owned)

	res, err := svc.Lookup(context.Background(), "becky chambers")
	require.NoError(t, err)

	// two series, alphabetical: "Monk & Robot" then "Wayfarers"
	require.Len(t, res.Series, 2)
	assert.Equal(t, "Monk & Robot", res.Series[0].Name)
	assert.Equal(t, "Wayfarers", res.Series[1].Name)

	// Wayfarers ordered by series position, not year-desc
	wayfarers := res.Series[1].Books
	assert.Equal(t, "The Long Way to a Small, Angry Planet", wayfarers[0].Title)
	assert.Equal(t, "Record of a Spaceborn Few", wayfarers[1].Title)

	// standalone kept separate
	require.Len(t, res.Standalone, 1)
	assert.Equal(t, "To Be Taught, If Fortunate", res.Standalone[0].Title)

	// flat = series books then standalone; index lines up with display
	require.Len(t, res.Flat, 4)
	assert.Equal(t, "A Psalm for the Wild-Built", res.Flat[0].Title)
	assert.Equal(t, "To Be Taught, If Fortunate", res.Flat[3].Title)

	// ownership: only the one in history is marked, fuzzy across punctuation/format
	assert.True(t, wayfarers[0].Owned, "title present in history should be owned")
	assert.False(t, wayfarers[1].Owned)
}

func TestSearchBooksMarksOwnedAndKeepsOrder(t *testing.T) {
	// relevance order from the provider, spanning multiple authors
	books := []dao.Book{
		{Title: "Why shoot a butler?", Authors: []string{"Georgette Heyer"}, Year: 1933},
		{Title: "Parable of the Sower", Authors: []string{"Octavia E. Butler"}, Year: 1993},
		{Title: "Kindred", Authors: []string{"Octavia E. Butler"}, Year: 1979},
	}

	owned := NewHistoryOwnedIndex([]string{"Octavia E. Butler - Kindred (epub)"})
	svc := NewLookupService(stubProvider{books: books}, owned)

	got, err := svc.SearchBooks(context.Background(), "butler")
	require.NoError(t, err)

	// order preserved (no series grouping / re-sort)
	require.Len(t, got, 3)
	assert.Equal(t, "Why shoot a butler?", got[0].Title)
	assert.Equal(t, "Kindred", got[2].Title)

	// only the history title is marked owned
	assert.False(t, got[0].Owned)
	assert.True(t, got[2].Owned, "title present in history should be owned")
}
