package entity

// bullshit go syntax for an enum
type IndexerType string

const (
	// name type = value
	IndexerTypeTorznab IndexerType = "TORZNAB"
)

// like the styling for structs in golang
type Indexer struct {
	ID          uint        `gorm:"primaryKey"`
	Name        string      `gorm:"size:255;uniqueIndex;not null"`
	BaseURL     string      `gorm:"size:1024;not null"`
	APIKey      string      `gorm:"size:255"`
	IndexerType IndexerType `gorm:"size:50;not null"`
	Enabled     bool        `gorm:"not null;default:true"`
}

func (Indexer) TableName() string {
	return "indexer"
}
