package entity

type LibraryConfig struct {
	ID          uint   `gorm:"primaryKey;check:id=1"`
	LibraryPath string `gorm:"size:1024;not null"`
	Template    string `gorm:"size:1024;not null"`
}

func (LibraryConfig) TableName() string {
	return "library_config"
}

const (
	DefaultLibraryPath = "~/lamplight"
	DefaultTemplate    = "{author}/{title} ({year})"
)
