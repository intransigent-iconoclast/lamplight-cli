package dao

type FilterCriteria struct {
	AllowedCategories []int // whitelist of Torznab category IDs
	// add more later e.g.:
	// MinSeeders        *int
	// MaxSizeBytes      *int64
}
