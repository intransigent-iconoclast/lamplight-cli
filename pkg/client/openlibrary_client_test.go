package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOpenLibrary returns a server plus a counter of requests made to
// /search/authors.json, so tests can assert LookupByKey never issues a name
// search.
func mockOpenLibrary(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	authorSearchHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/authors.json"):
			authorSearchHits++
			// three ranked candidates, in OpenLibrary's (non-work-count-sorted)
			// order: Becky Chambers has the highest work_count so she remains
			// the best-match pick for name-based Lookup.
			_, _ = w.Write([]byte(`{"docs":[
				{"key":"OL1A","name":"Becky Chambers Jr.","work_count":2},
				{"key":"OL42A","name":"Becky Chambers","work_count":6},
				{"key":"OL2A","name":"Rebecca Chambers","work_count":4}
			]}`))
		case strings.HasPrefix(r.URL.Path, "/authors/"):
			_, _ = w.Write([]byte(`{
				"name":"Becky Chambers",
				"bio":{"value":"American SF author. Known for Wayfarers."},
				"birth_date":"3 May 1985",
				"photos":[-1,987,988],
				"alternate_names":["B. Chambers"],
				"links":[{"title":"Official Website","url":"https://example.com"}],
				"remote_ids":{"goodreads":"17650479","wikidata":"Q25298820"}
			}`))
		case strings.HasPrefix(r.URL.Path, "/search.json"):
			_, _ = w.Write([]byte(`{"docs":[
				{"key":"/works/OL1W","title":"The Long Way to a Small, Angry Planet","first_publish_year":2014,"cover_i":111,"isbn":["9781473619814","1473619815"],"ratings_average":4.3,"author_name":["Becky Chambers"]},
				{"key":"/works/OL2W","title":"A Closed and Common Orbit","first_publish_year":2016,"ratings_average":4.2,"author_name":["Becky Chambers"]},
				{"key":"/works/OL1Wdup","title":"the long way to a small, angry planet","first_publish_year":2015,"author_name":["Becky Chambers"]}
			]}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	return srv, &authorSearchHits
}

func TestOpenLibraryLookup(t *testing.T) {
	srv, _ := mockOpenLibrary(t)
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	author, err := c.Lookup(context.Background(), "becky chambers")
	require.NoError(t, err)

	assert.Equal(t, "Becky Chambers", author.Name)
	assert.Equal(t, 6, author.WorkCount)

	// full bio now (renderers truncate; frontend gets the whole thing)
	assert.Equal(t, "American SF author. Known for Wayfarers.", author.Bio)
	assert.Equal(t, "3 May 1985", author.BirthDate)
	assert.Equal(t, []string{"B. Chambers"}, author.AlternateNames)
	assert.Equal(t, "17650479", author.RemoteIDs["goodreads"])
	require.Len(t, author.Links, 1)
	assert.Equal(t, "https://example.com", author.Links[0].URL)

	// first positive photo id wins (-1 is OpenLibrary's "no photo" sentinel)
	assert.Equal(t, 987, author.PhotoID)
	assert.Contains(t, author.PhotoURL, "/a/id/987-L.jpg")

	require.Len(t, author.Books, 2, "duplicate title should be collapsed")

	// sorted newest-first
	assert.Equal(t, "A Closed and Common Orbit", author.Books[0].Title)
	assert.Equal(t, 2016, author.Books[0].Year)

	tlw := author.Books[1]
	assert.Equal(t, 2014, tlw.Year)
	assert.Equal(t, 4.3, tlw.Rating)
	assert.Equal(t, "9781473619814", tlw.ISBN, "ISBN-13 preferred")
	assert.Equal(t, 111, tlw.CoverID)
	assert.Contains(t, tlw.CoverURL, "/b/id/111-M.jpg")
}

func TestOpenLibrarySearchAuthors(t *testing.T) {
	srv, hits := mockOpenLibrary(t)
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	authors, err := c.SearchAuthors(context.Background(), "chambers")
	require.NoError(t, err)
	require.Len(t, authors, 3)

	// preserves OpenLibrary's ranking order — not re-sorted by work count
	assert.Equal(t, "Becky Chambers Jr.", authors[0].Name)
	assert.Equal(t, "Becky Chambers", authors[1].Name)
	assert.Equal(t, "Rebecca Chambers", authors[2].Name)

	assert.Equal(t, "OL42A", authors[1].Key)
	assert.Equal(t, 6, authors[1].WorkCount)
	assert.Contains(t, authors[1].PhotoURL, "/a/olid/OL42A-M.jpg")

	// no enrichment — no per-author detail call
	assert.Empty(t, authors[1].Bio)
	assert.Empty(t, authors[1].Links)

	assert.Equal(t, 1, *hits, "SearchAuthors should issue exactly one author search request")
}

func TestOpenLibraryLookupByKey(t *testing.T) {
	srv, hits := mockOpenLibrary(t)
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	author, err := c.LookupByKey(context.Background(), "OL42A")
	require.NoError(t, err)

	// Name comes from the detail record — LookupByKey has no search step to get it from
	assert.Equal(t, "Becky Chambers", author.Name)
	// enriched, same as Lookup
	assert.Equal(t, "American SF author. Known for Wayfarers.", author.Bio)
	assert.Equal(t, "3 May 1985", author.BirthDate)
	require.Len(t, author.Books, 2, "duplicate title should be collapsed")

	// resolved directly by key — no author name search
	assert.Equal(t, 0, *hits, "LookupByKey must not issue an author name search")
}

func TestOpenLibraryLookupByKeyEmpty(t *testing.T) {
	srv, _ := mockOpenLibrary(t)
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.LookupByKey(context.Background(), "")
	require.Error(t, err)
}

func TestOpenLibraryLookupNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"docs":[]}`))
	}))
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.Lookup(context.Background(), "nobody at all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no author found")
}
