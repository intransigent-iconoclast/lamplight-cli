package dao

type SearchResult struct {
	Title       string
	Link        string
	IndexerName string
	SizeBytes   *int64 // * somehow makes this an optional
	Seeders     *int
	Leechers    *int
	Categories  []int
	FormatAttr  string // raw torznab:attr name="format" value, e.g. "epub", "pdf"
	Format      string // resolved display format (epub/pdf/mobi/audiobook/comic/unknown)
}
