package dao

type SearchRequest struct {
	Query       string
	IndexerName string
	Limit       int // default maybe -1 idk for now lets roll with it
}
