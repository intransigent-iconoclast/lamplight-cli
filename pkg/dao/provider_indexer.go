package dao

type ProviderIndexerDAO struct {
	Name       string
	ExternalID string
	Caps       []int // supported categories from the indexer response
}
