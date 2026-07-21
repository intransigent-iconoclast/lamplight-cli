package entity

import "time"

// This represents a singleton table with only one value.
type SearchCache struct {
	ID        uint   `gorm:"primaryKey;check:id=1"`
	Result    string `gorm:"type:text;not null"` // json blob cached results
	CreatedAt time.Time
	UpdatedAt time.Time
}
