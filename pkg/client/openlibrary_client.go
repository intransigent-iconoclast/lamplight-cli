package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/intransigent-iconoclast/lamplight-cli/pkg/dao"
)

const (
	OpenLibraryBaseURL = "https://openlibrary.org"
	coversBaseURL      = "https://covers.openlibrary.org/b/id"
)

type OpenLibraryClient struct {
	Client  *http.Client
	BaseURL string
}

func NewOpenLibraryClient(client *http.Client) *OpenLibraryClient {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &OpenLibraryClient{
		Client:  client,
		BaseURL: OpenLibraryBaseURL,
	}
}

type olAuthorSearchResponse struct {
	Docs []struct {
		Key       string `json:"key"`
		Name      string `json:"name"`
		WorkCount int    `json:"work_count"`
	} `json:"docs"`
}

type olAuthorDetail struct {
	Bio json.RawMessage `json:"bio"`
}

type olWorksSearchResponse struct {
	Docs []struct {
		Key            string   `json:"key"`
		Title          string   `json:"title"`
		FirstPublished int      `json:"first_publish_year"`
		CoverID        int      `json:"cover_i"`
		ISBN           []string `json:"isbn"`
		RatingsAverage float64  `json:"ratings_average"`
		AuthorName     []string `json:"author_name"`
	} `json:"docs"`
}

func (c *OpenLibraryClient) Lookup(ctx context.Context, query string) (*dao.Author, error) {
	author, err := c.searchAuthor(ctx, query)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, fmt.Errorf("no author found matching %q", query)
	}

	if bio, err := c.authorBio(ctx, author.Key); err == nil {
		author.Bio = bio
	}

	books, err := c.works(ctx, author.Key)
	if err != nil {
		return nil, err
	}
	author.Books = books
	return author, nil
}

func (c *OpenLibraryClient) searchAuthor(ctx context.Context, query string) (*dao.Author, error) {
	u := fmt.Sprintf("%s/search/authors.json?q=%s", c.BaseURL, url.QueryEscape(query))

	var resp olAuthorSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, fmt.Errorf("author search: %w", err)
	}
	if len(resp.Docs) == 0 {
		return nil, nil
	}

	// Results aren't popularity-ranked, so the first hit is often a near-namesake.
	// The author with the most works is almost always the one meant.
	best := resp.Docs[0]
	for _, d := range resp.Docs[1:] {
		if d.WorkCount > best.WorkCount {
			best = d
		}
	}
	return &dao.Author{Name: best.Name, Key: best.Key}, nil
}

func (c *OpenLibraryClient) authorBio(ctx context.Context, key string) (string, error) {
	u := fmt.Sprintf("%s/authors/%s.json", c.BaseURL, url.PathEscape(key))

	var detail olAuthorDetail
	if err := c.getJSON(ctx, u, &detail); err != nil {
		return "", err
	}
	if len(detail.Bio) == 0 {
		return "", nil
	}

	// bio is either a bare string or {"value": "..."}
	var asString string
	if err := json.Unmarshal(detail.Bio, &asString); err == nil {
		return firstSentence(asString), nil
	}
	var asObject struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(detail.Bio, &asObject); err == nil {
		return firstSentence(asObject.Value), nil
	}
	return "", nil
}

// workFields is the shared field set for the search.json endpoint: enough to
// build a dao.Book (title/author/year/rating/cover/isbn) while keeping payloads
// small.
const workFields = "key,title,first_publish_year,cover_i,isbn,ratings_average,author_name"

// works lists a single author's catalog. Filtering on author_key (not a fuzzy
// name match) keeps out namesakes' books while still returning ratings/covers/
// year, which the works.json endpoint omits. Ordered newest-first.
func (c *OpenLibraryClient) works(ctx context.Context, authorKey string) ([]dao.Book, error) {
	u := fmt.Sprintf("%s/search.json?author_key=%s&fields=%s&limit=100",
		c.BaseURL, url.QueryEscape(authorKey), url.QueryEscape(workFields))

	var resp olWorksSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, fmt.Errorf("works search: %w", err)
	}

	books := docsToBooks(resp)
	sort.SliceStable(books, func(i, j int) bool {
		return books[i].Year > books[j].Year
	})
	return books, nil
}

// SearchBooks does a general catalog search (title, author, or keyword) against
// the same search.json endpoint. Unlike works, results span many authors and
// stay in OpenLibrary's relevance order — so callers should surface author_name
// per row, since a common-word query can blend title- and author-matches.
func (c *OpenLibraryClient) SearchBooks(ctx context.Context, query string) ([]dao.Book, error) {
	u := fmt.Sprintf("%s/search.json?q=%s&fields=%s&limit=25",
		c.BaseURL, url.QueryEscape(query), url.QueryEscape(workFields))

	var resp olWorksSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, fmt.Errorf("book search: %w", err)
	}
	return docsToBooks(resp), nil
}

// docsToBooks maps a search.json response to books, dropping untitled docs and
// de-duplicating by normalized title while preserving input order.
func docsToBooks(resp olWorksSearchResponse) []dao.Book {
	books := make([]dao.Book, 0, len(resp.Docs))
	seen := make(map[string]bool)
	for _, d := range resp.Docs {
		if d.Title == "" {
			continue
		}
		norm := strings.ToLower(strings.TrimSpace(d.Title))
		if seen[norm] {
			continue
		}
		seen[norm] = true

		b := dao.Book{
			Title:   d.Title,
			Authors: d.AuthorName,
			Year:    d.FirstPublished,
			Rating:  d.RatingsAverage,
			WorkKey: d.Key,
		}
		if len(d.ISBN) > 0 {
			b.ISBN = preferISBN13(d.ISBN)
		}
		if d.CoverID > 0 {
			b.CoverURL = fmt.Sprintf("%s/%d-M.jpg", coversBaseURL, d.CoverID)
		}
		books = append(books, b)
	}
	return books
}

func (c *OpenLibraryClient) getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "lamplight-cli (https://github.com/intransigent-iconoclast/lamplight-cli)")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func preferISBN13(isbns []string) string {
	for _, s := range isbns {
		if len(s) == 13 {
			return s
		}
	}
	return isbns[0]
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '.'); i > 0 {
		return s[:i+1]
	}
	return s
}
