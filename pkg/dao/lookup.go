package dao

// Book is a work from OpenLibrary, enriched with everything the frontend might
// render. The CLI renders only a minimal subset; the rest is here so the web UI
// (and --json consumers) can show covers, ratings, subjects, read-online links,
// etc. without extra round-trips.
type Book struct {
	Title      string
	Authors    []string
	AuthorKeys []string // OpenLibrary author keys, aligned with Authors — for linking to author pages
	Year       int      // first publish year

	ISBN     string // preferred edition ISBN (ISBN-13 when available)
	CoverID  int    // OpenLibrary cover id; 0 = none. Build any size URL from it (…/b/id/<id>-{S,M,L}.jpg).
	CoverURL string // convenience medium cover URL ("" = none)

	Rating       float64
	RatingsCount int
	EditionCount int
	Pages        int      // median page count across editions
	Languages    []string // ISO 639-2 codes, e.g. "eng"
	Subjects     []string // most-relevant first, capped

	FirstSentence string
	EbookAccess   string // "public" | "borrowable" | "printdisabled" | "no_ebook"
	HasFulltext   bool
	ArchiveID     string // Internet Archive id (first), for a read-online link

	SeriesName string
	SeriesPos  float64

	Owned   bool
	WorkKey string
}

// AuthorLink is an external link on an author's OpenLibrary record (official
// site, ISFDB, etc.).
type AuthorLink struct {
	Title string
	URL   string
}

// Author is an OpenLibrary author enriched for rich rendering. Bio is the full
// text (renderers truncate as needed); Photo/Links/RemoteIDs let the frontend
// show a portrait and link out.
type Author struct {
	Name string
	Key  string
	Bio  string

	PhotoID  int    // OpenLibrary photo id; 0 = none. Build any size from …/a/id/<id>-{S,M,L}.jpg.
	PhotoURL string // convenience large photo URL ("" = none)

	BirthDate      string
	DeathDate      string
	AlternateNames []string
	Links          []AuthorLink
	RemoteIDs      map[string]string // e.g. goodreads, wikidata, storygraph, isni

	WorkCount int
	Books     []Book
}
