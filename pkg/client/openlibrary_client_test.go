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

func mockOpenLibrary(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/authors.json"):
			w.Write([]byte(`{"docs":[{"key":"OL42A","name":"Becky Chambers","work_count":6}]}`))
		case strings.HasPrefix(r.URL.Path, "/authors/"):
			w.Write([]byte(`{
				"bio":{"value":"American SF author. Known for Wayfarers."},
				"birth_date":"3 May 1985",
				"photos":[-1,987,988],
				"alternate_names":["B. Chambers"],
				"links":[{"title":"Official Website","url":"https://example.com"}],
				"remote_ids":{"goodreads":"17650479","wikidata":"Q25298820"}
			}`))
		case strings.HasPrefix(r.URL.Path, "/search.json"):
			w.Write([]byte(`{"docs":[
				{"key":"/works/OL1W","title":"The Long Way to a Small, Angry Planet","first_publish_year":2014,"cover_i":111,"isbn":["9781473619814","1473619815"],"ratings_average":4.3,"author_name":["Becky Chambers"]},
				{"key":"/works/OL2W","title":"A Closed and Common Orbit","first_publish_year":2016,"ratings_average":4.2,"author_name":["Becky Chambers"]},
				{"key":"/works/OL1Wdup","title":"the long way to a small, angry planet","first_publish_year":2015,"author_name":["Becky Chambers"]}
			]}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
}

func TestOpenLibraryLookup(t *testing.T) {
	srv := mockOpenLibrary(t)
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

func TestOpenLibraryLookupNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"docs":[]}`))
	}))
	defer srv.Close()

	c := NewOpenLibraryClient(srv.Client())
	c.BaseURL = srv.URL

	_, err := c.Lookup(context.Background(), "nobody at all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no author found")
}
