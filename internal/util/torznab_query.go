package utils

import (
	"net/url"
	"strconv"
	"strings"
)

// bullshit go syntax for an enum
type SearchType string

const (
	// name type = value
	SEARCH SearchType = "search"
	BOOK   SearchType = "book"
)

type TorznabQuery struct {
	SearchType SearchType
	Query      string
	Title      string
	Author     string
	Isbn       string
	Categories []int
	Limit      *int // * makes them optional somehow when they're pointers
	Offset     *int
}

func NewTorznabQuery() TorznabQuery {
	return TorznabQuery{
		SearchType: SEARCH,
	}
}

func (q TorznabQuery) ToParams(apiKey string) url.Values {
	params := url.Values{}

	params.Set("apikey", apiKey)

	searchType := q.SearchType
	if searchType == "" {
		searchType = SEARCH
	}
	params.Set("t", string(searchType))

	if s := strings.TrimSpace(q.Query); s != "" {
		params.Set("q", s)
	}
	if s := strings.TrimSpace(q.Title); s != "" {
		params.Set("title", s)
	}
	if s := strings.TrimSpace(q.Author); s != "" {
		params.Set("author", s)
	}
	if s := strings.TrimSpace(q.Isbn); s != "" {
		params.Set("isbn", s)
	}

	if len(q.Categories) > 0 {
		// this is a neat go thing chatgpt taught me this is a set constructed from a map
		// format is map[key]value and why would u use struct{}? well because struct{} uses 0 bytes
		// we don't have any values to store so we just need keys
		seen := make(map[int]struct{})
		// acumulate the string values of categories here
		var cats []string
		for _, c := range q.Categories {
			// refer here for how to use sets in golang https://www.willem.dev/articles/sets-in-golang/
			if _, ok := seen[c]; ok {
				continue
			}
			// add string to our collection
			cats = append(cats, strconv.Itoa(c))
			// add a key but don't add a value ...
			seen[c] = struct{}{}
		}
		if len(cats) > 0 {
			params.Set("cat", strings.Join(cats, ","))
		}
	}
	if q.Limit != nil {
		params.Set("limit", strconv.Itoa(*q.Limit))
	}
	if q.Offset != nil {
		params.Set("offset", strconv.Itoa(*q.Offset))
	}
	return params
}
