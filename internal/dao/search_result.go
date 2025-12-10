package dao

type SearchResult struct {
	Title       string
	Link        string
	IndexerName string
	SizeBytes   *int64 // * somehow makes this an optional
	Seeders     *int
	Leechers    *int
	Categories  []int
}
