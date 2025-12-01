package dao

type SearchRequest struct {
	Query       string
	Author      string
	Title       string
	ISBN        string
	IndexerName string
	Limit       int
}
