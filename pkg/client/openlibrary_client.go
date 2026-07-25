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
	coversBooksBase    = "https://covers.openlibrary.org/b/id"
	coversAuthorsBase  = "https://covers.openlibrary.org/a/id"
	// coversAuthorsOLIDBase builds a photo URL directly from an author's
	// OpenLibrary key (OLID), with no numeric photo id required — used by
	// SearchAuthors, which doesn't fetch the detail record.
	coversAuthorsOLIDBase = "https://covers.openlibrary.org/a/olid"
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

type olAuthorDoc struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	WorkCount int    `json:"work_count"`
}

type olAuthorSearchResponse struct {
	Docs []olAuthorDoc `json:"docs"`
}

type olAuthorDetail struct {
	Name           string          `json:"name"`
	Bio            json.RawMessage `json:"bio"`
	Photos         []int           `json:"photos"`
	BirthDate      string          `json:"birth_date"`
	DeathDate      string          `json:"death_date"`
	AlternateNames []string        `json:"alternate_names"`
	Links          []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"links"`
	RemoteIDs map[string]string `json:"remote_ids"`
}

type olWorksSearchResponse struct {
	Docs []olWorkDoc `json:"docs"`
}

type olWorkDoc struct {
	Key            string   `json:"key"`
	Title          string   `json:"title"`
	FirstPublished int      `json:"first_publish_year"`
	CoverID        int      `json:"cover_i"`
	ISBN           []string `json:"isbn"`
	RatingsAverage float64  `json:"ratings_average"`
	RatingsCount   int      `json:"ratings_count"`
	EditionCount   int      `json:"edition_count"`
	Pages          int      `json:"number_of_pages_median"`
	AuthorName     []string `json:"author_name"`
	AuthorKey      []string `json:"author_key"`
	Language       []string `json:"language"`
	Subject        []string `json:"subject"`
	FirstSentence  []string `json:"first_sentence"`
	EbookAccess    string   `json:"ebook_access"`
	HasFulltext    bool     `json:"has_fulltext"`
	IA             []string `json:"ia"`
}

func (c *OpenLibraryClient) Lookup(ctx context.Context, query string) (*dao.Author, error) {
	author, err := c.searchAuthor(ctx, query)
	if err != nil {
		return nil, err
	}
	if author == nil {
		return nil, fmt.Errorf("no author found matching %q", query)
	}
	if err := c.hydrate(ctx, author); err != nil {
		return nil, err
	}
	return author, nil
}

// LookupByKey builds a full author record (enrichment + works) from a known
// OpenLibrary author key, skipping the ambiguous name search entirely. Use this
// when the caller already has a specific author's key (e.g. from SearchAuthors
// or a stored reference) and wants that exact author, not a best-name-match.
func (c *OpenLibraryClient) LookupByKey(ctx context.Context, key string) (*dao.Author, error) {
	if strings.TrimSpace(key) == "" {
		return nil, fmt.Errorf("author key is required")
	}
	author := &dao.Author{Key: key}
	if err := c.hydrate(ctx, author); err != nil {
		return nil, err
	}
	return author, nil
}

// hydrate enriches a (bio, photo, dates, links) and attaches its works. a.Key
// must already be set. Enrichment is best-effort: a missing or partial detail
// record shouldn't sink an otherwise-good bibliography.
func (c *OpenLibraryClient) hydrate(ctx context.Context, a *dao.Author) error {
	_ = c.enrichAuthor(ctx, a)

	books, err := c.works(ctx, a.Key)
	if err != nil {
		return err
	}
	a.Books = books
	return nil
}

// authorSearchDocs performs the /search/authors.json request and returns the
// raw candidate docs in OpenLibrary's ranking order.
func (c *OpenLibraryClient) authorSearchDocs(ctx context.Context, query string) ([]olAuthorDoc, error) {
	u := fmt.Sprintf("%s/search/authors.json?q=%s", c.BaseURL, url.QueryEscape(query))

	var resp olAuthorSearchResponse
	if err := c.getJSON(ctx, u, &resp); err != nil {
		return nil, fmt.Errorf("author search: %w", err)
	}
	return resp.Docs, nil
}

func (c *OpenLibraryClient) searchAuthor(ctx context.Context, query string) (*dao.Author, error) {
	docs, err := c.authorSearchDocs(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, nil
	}

	// Results aren't popularity-ranked, so the first hit is often a near-namesake.
	// The author with the most works is almost always the one meant.
	best := docs[0]
	for _, d := range docs[1:] {
		if d.WorkCount > best.WorkCount {
			best = d
		}
	}
	return &dao.Author{Name: best.Name, Key: best.Key, WorkCount: best.WorkCount}, nil
}

// SearchAuthors returns every author candidate matching query, in
// OpenLibrary's ranking order, without the per-author detail call that
// enrichAuthor performs (no bio/links/dates — that would be one extra HTTP
// round-trip per candidate). PhotoURL is built directly from the OpenLibrary
// author key (OLID), so it costs nothing extra to include.
func (c *OpenLibraryClient) SearchAuthors(ctx context.Context, query string) ([]dao.Author, error) {
	docs, err := c.authorSearchDocs(ctx, query)
	if err != nil {
		return nil, err
	}

	authors := make([]dao.Author, 0, len(docs))
	for _, d := range docs {
		a := dao.Author{Name: d.Name, Key: d.Key, WorkCount: d.WorkCount}
		if d.Key != "" {
			a.PhotoURL = fmt.Sprintf("%s/%s-M.jpg", coversAuthorsOLIDBase, d.Key)
		}
		authors = append(authors, a)
	}
	return authors, nil
}

// enrichAuthor fills in bio, portrait, dates, links, and remote ids from the
// author's detail record. It mutates a in place; callers treat failures as
// non-fatal.
func (c *OpenLibraryClient) enrichAuthor(ctx context.Context, a *dao.Author) error {
	u := fmt.Sprintf("%s/authors/%s.json", c.BaseURL, url.PathEscape(a.Key))

	var d olAuthorDetail
	if err := c.getJSON(ctx, u, &d); err != nil {
		return err
	}

	// LookupByKey has no name (it skips the name search entirely); the detail
	// record is authoritative, so fill it in when the caller didn't already
	// have one from a search step.
	if a.Name == "" && d.Name != "" {
		a.Name = d.Name
	}
	a.Bio = bioText(d.Bio)
	a.BirthDate = d.BirthDate
	a.DeathDate = d.DeathDate
	a.AlternateNames = d.AlternateNames
	a.RemoteIDs = d.RemoteIDs
	for _, l := range d.Links {
		a.Links = append(a.Links, dao.AuthorLink{Title: l.Title, URL: l.URL})
	}
	if id := firstPositive(d.Photos); id > 0 {
		a.PhotoID = id
		a.PhotoURL = fmt.Sprintf("%s/%d-L.jpg", coversAuthorsBase, id)
	}
	return nil
}

// workFields is the shared field set for the search.json endpoint: enough to
// build a dao.Book (title/author/year/rating/cover/isbn) while keeping payloads
// small.
const workFields = "key,title,first_publish_year,cover_i,isbn,ratings_average,ratings_count," +
	"edition_count,number_of_pages_median,author_name,author_key,language,subject," +
	"first_sentence,ebook_access,has_fulltext,ia"

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
			Title:        d.Title,
			Authors:      d.AuthorName,
			AuthorKeys:   d.AuthorKey,
			Year:         d.FirstPublished,
			Rating:       d.RatingsAverage,
			RatingsCount: d.RatingsCount,
			EditionCount: d.EditionCount,
			Pages:        d.Pages,
			Languages:    d.Language,
			Subjects:     capStrings(d.Subject, 12),
			EbookAccess:  d.EbookAccess,
			HasFulltext:  d.HasFulltext,
			WorkKey:      d.Key,
		}
		if len(d.ISBN) > 0 {
			b.ISBN = preferISBN13(d.ISBN)
		}
		if len(d.FirstSentence) > 0 {
			b.FirstSentence = d.FirstSentence[0]
		}
		if len(d.IA) > 0 {
			b.ArchiveID = d.IA[0]
		}
		if d.CoverID > 0 {
			b.CoverID = d.CoverID
			b.CoverURL = fmt.Sprintf("%s/%d-M.jpg", coversBooksBase, d.CoverID)
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

// bioText normalizes OpenLibrary's bio field, which is either a bare string or
// a {"value": "..."} object, into the full bio text.
func bioText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	var asObject struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &asObject); err == nil {
		return strings.TrimSpace(asObject.Value)
	}
	return ""
}

// firstPositive returns the first positive id (OpenLibrary uses -1 as a
// "no photo" sentinel), or 0 if none.
func firstPositive(ids []int) int {
	for _, id := range ids {
		if id > 0 {
			return id
		}
	}
	return 0
}

// capStrings returns at most n elements, trimming noisy long lists (subjects can
// run to hundreds).
func capStrings(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
