package entity

import "time"

// Singleton row holding the last lookup's flattened books for `lookup --get N`.
type LookupCache struct {
	ID        uint   `gorm:"primaryKey;check:id=1"`
	Result    string `gorm:"type:text;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (LookupCache) TableName() string {
	return "lookup_cache"
}
